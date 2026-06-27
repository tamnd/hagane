package emit

import (
	"bytes"
	"fmt"
	"go/token"
	"go/types"
	"sort"
	"strings"

	"golang.org/x/tools/go/ssa"
)

// pkgEmitter emits one package into e.hdrbuf (declarations) and e.buf (bodies).
type pkgEmitter struct {
	e             *Emitter
	pkg           *ssa.Package
	inRecoverBlk  bool               // true while emitting fn.Recover block
	curFnHasFrame bool               // true if current function has a setjmp panic frame
	earlyAllocs   map[ssa.Value]bool // Allocs hoisted before the setjmp frame
}

func (pe *pkgEmitter) prefix() string {
	return pkgCPrefix(pe.pkg.Pkg.Path())
}

// emitHeader emits struct typedefs, global var declarations, and function prototypes into e.hdrbuf.
func (pe *pkgEmitter) emitHeader() {
	h := pe.e.hdrbuf

	// emit global variable declarations
	var names []string
	for name := range pe.pkg.Members {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		g, ok := pe.pkg.Members[name].(*ssa.Global)
		if !ok {
			continue
		}
		ct := pe.e.cTypeOf(g.Type())
		// g.Type() is always *T; emit the T
		var elemCT string
		if ptr, ok := g.Type().Underlying().(*types.Pointer); ok {
			elemCT = pe.e.cTypeOf(ptr.Elem())
		} else {
			elemCT = ct
		}
		cname := pe.prefix() + sanitizeName(g.Name())
		fmt.Fprintf(h, "extern %s %s;\n", elemCT, cname)
	}

	// Collect named types in sorted order for deterministic output.
	var typeNames []string
	for name, mem := range pe.pkg.Members {
		if _, ok := mem.(*ssa.Type); ok {
			typeNames = append(typeNames, name)
		}
	}
	sort.Strings(typeNames)

	// Forward-declare all struct types first so self-referential or mutually
	// recursive structs can reference each other in their field lists.
	for _, name := range typeNames {
		tp := pe.pkg.Members[name].(*ssa.Type)
		named, ok := tp.Type().(*types.Named)
		if !ok {
			continue
		}
		if _, ok := named.Underlying().(*types.Struct); ok {
			cname := pe.e.cTypeNamed(named)
			fmt.Fprintf(h, "typedef struct %s %s;\n", cname, cname)
		}
	}

	// Emit non-struct type aliases first (slices, basics, funcs, etc.) so that
	// struct bodies that reference these aliases see them already defined.
	for _, name := range typeNames {
		tp := pe.pkg.Members[name].(*ssa.Type)
		named, ok := tp.Type().(*types.Named)
		if !ok {
			continue
		}
		if _, ok := named.Underlying().(*types.Struct); ok {
			continue // handled in the next pass
		}
		cname := pe.e.cTypeNamed(named)
		underlying := pe.e.cTypeInner(named.Underlying())
		fmt.Fprintf(h, "typedef %s %s;\n", underlying, cname)
	}

	// Emit struct bodies second (after all aliases and slice typedefs are defined).
	// We buffer each struct body separately because cTypeOf() may write additional
	// slice typedefs to hdrbuf as a side effect — those must land before the body,
	// not inside it.
	for _, name := range typeNames {
		tp := pe.pkg.Members[name].(*ssa.Type)
		named, ok := tp.Type().(*types.Named)
		if !ok {
			continue
		}
		st, ok := named.Underlying().(*types.Struct)
		if !ok {
			continue
		}
		cname := pe.e.cTypeNamed(named)
		var bodyBuf bytes.Buffer
		fmt.Fprintf(&bodyBuf, "struct %s {\n", cname)
		for i := 0; i < st.NumFields(); i++ {
			f := st.Field(i)
			// cTypeOf may write slice typedefs to hdrbuf (h) here — that is intentional.
			// Those writes land before bodyBuf is flushed, ensuring correct C ordering.
			ft := pe.e.cTypeOf(f.Type())
			fmt.Fprintf(&bodyBuf, "    %s %s;\n", ft, f.Name())
		}
		fmt.Fprintf(&bodyBuf, "};\n")
		h.Write(bodyBuf.Bytes())
	}

	// Collect function members in sorted order for deterministic output.
	var fnNames []string
	for name, mem := range pe.pkg.Members {
		if fn, ok := mem.(*ssa.Function); ok && fn.Blocks != nil {
			fnNames = append(fnNames, name)
		}
	}
	sort.Strings(fnNames)

	// Emit multi-return struct typedefs before function prototypes.
	for _, name := range fnNames {
		fn := pe.pkg.Members[name].(*ssa.Function)
		if fn.Signature.Results().Len() > 1 {
			decl := retStructDecl(pe.e, pe.prefix(), methodCBaseName(fn), fn.Signature.Results())
			fmt.Fprintf(h, "%s\n", decl)
		}
	}

	// Emit function prototypes.
	for _, name := range fnNames {
		fn := pe.pkg.Members[name].(*ssa.Function)
		fmt.Fprintf(h, "%s;\n", pe.funcSignature(fn))
	}
}

// methodCBaseName returns the C function base name for fn, including receiver type
// for methods so that English.Greet and Spanish.Greet don't collide.
func methodCBaseName(fn *ssa.Function) string {
	recv := fn.Signature.Recv()
	if recv == nil {
		return sanitizeName(fn.Name())
	}
	// extract the receiver type name (dereference pointer receiver)
	t := recv.Type()
	if ptr, ok := t.(*types.Pointer); ok {
		t = ptr.Elem()
	}
	if named, ok := t.(*types.Named); ok {
		return sanitizeName(named.Obj().Name()) + "_" + sanitizeName(fn.Name())
	}
	return sanitizeName(fn.Name())
}

func (pe *pkgEmitter) funcSignature(fn *ssa.Function) string {
	cname := pe.prefix() + methodCBaseName(fn)
	retType := pe.returnCType(fn)

	var params []string
	// closures get an extra env pointer as first parameter
	if len(fn.FreeVars) > 0 {
		params = append(params, "void *_hg_env")
	}
	for i, p := range fn.Params {
		ct := pe.e.cTypeOf(p.Type())
		pname := paramName(p, i)
		params = append(params, ct+" "+pname)
	}
	if len(params) == 0 {
		params = []string{"void"}
	}
	return fmt.Sprintf("%s %s(%s)", retType, cname, strings.Join(params, ", "))
}

// closureEnvTypeName returns the C env struct type name for an anon func.
func (pe *pkgEmitter) closureEnvTypeName(fn *ssa.Function) string {
	return pe.prefix() + sanitizeName(fn.Name()) + "_env_t"
}

// emitClosureEnv emits the env struct typedef for fn (if it has free vars)
// into the header buffer, and returns the type name.
func (pe *pkgEmitter) emitClosureEnv(fn *ssa.Function) string {
	if len(fn.FreeVars) == 0 {
		return ""
	}
	name := pe.closureEnvTypeName(fn)
	if pe.e.mapFuncs[name] { // reuse mapFuncs as general "already emitted" set
		return name
	}
	pe.e.mapFuncs[name] = true
	// Compute field C types BEFORE opening the struct body.
	// cTypeOf may write side effects (anonymous struct typedefs, slice typedefs)
	// to hdrbuf; those writes must appear before the typedef struct { ... } body.
	type envField struct{ ct, nm string }
	fields := make([]envField, len(fn.FreeVars))
	for i, fv := range fn.FreeVars {
		fields[i] = envField{pe.e.cTypeOf(fv.Type()), sanitizeName(fv.Name())}
	}
	fmt.Fprintf(pe.e.hdrbuf, "typedef struct {\n")
	for _, f := range fields {
		fmt.Fprintf(pe.e.hdrbuf, "    %s %s;\n", f.ct, f.nm)
	}
	fmt.Fprintf(pe.e.hdrbuf, "} %s;\n", name)
	return name
}

func (pe *pkgEmitter) returnCType(fn *ssa.Function) string {
	results := fn.Signature.Results()
	switch results.Len() {
	case 0:
		return "void"
	case 1:
		return pe.e.cTypeOf(results.At(0).Type())
	default:
		return retStructName(pe.prefix(), sanitizeName(fn.Name()))
	}
}

// emitGlobalDefs emits global variable definitions (T name;) into the body buffer.
// The header only carries extern declarations; this function provides the single
// definition that the linker requires, placed in the .c file so it appears once.
func (pe *pkgEmitter) emitGlobalDefs() {
	b := pe.e.buf
	var names []string
	for name := range pe.pkg.Members {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		g, ok := pe.pkg.Members[name].(*ssa.Global)
		if !ok {
			continue
		}
		var elemCT string
		if ptr, ok := g.Type().Underlying().(*types.Pointer); ok {
			elemCT = pe.e.cTypeOf(ptr.Elem())
		} else {
			elemCT = pe.e.cTypeOf(g.Type())
		}
		cname := pe.prefix() + sanitizeName(g.Name())
		fmt.Fprintf(b, "%s %s;\n", elemCT, cname)
	}
	if len(names) > 0 {
		fmt.Fprintf(b, "\n")
	}
}

// emitFunc emits a complete C function definition into e.buf.
func (pe *pkgEmitter) emitFunc(fn *ssa.Function) {
	b := pe.e.buf

	pe.emitNeededSliceTypes(fn)

	// emit env struct typedef for closures (into header, before signature)
	envTypeName := pe.emitClosureEnv(fn)

	fmt.Fprintf(b, "%s {\n", pe.funcSignature(fn))

	// unpack env pointer into typed struct for closures
	if envTypeName != "" {
		fmt.Fprintf(b, "    %s *_env = (%s*)_hg_env;\n", envTypeName, envTypeName)
	}

	// defer linked-list head + optional setjmp panic frame
	hasDefers := pe.fnHasDefers(fn)
	pe.curFnHasFrame = hasDefers && fn.Recover != nil
	pe.earlyAllocs = nil
	if hasDefers {
		if pe.curFnHasFrame {
			// volatile so -O2 doesn't cache it in a register across the setjmp
			fmt.Fprintf(b, "    hg_defer_t * volatile _hg_defer_head = NULL;\n")
		} else {
			fmt.Fprintf(b, "    hg_defer_t *_hg_defer_head = NULL;\n")
		}
	}
	if pe.curFnHasFrame {
		// Named return variables are loaded in the recover block.  Their Allocs live
		// in blk0.  If panic fires before blk0 runs them, the recover block would
		// dereference uninitialised pointers.  Emit those Allocs now — before the
		// setjmp — so they are always valid when the recover block runs.
		pe.earlyAllocs = map[ssa.Value]bool{}
		for _, instr := range fn.Recover.Instrs {
			unop, ok := instr.(*ssa.UnOp)
			if !ok {
				continue
			}
			alloc, ok := unop.X.(*ssa.Alloc)
			if !ok {
				continue
			}
			if pe.earlyAllocs[alloc] {
				continue
			}
			pe.earlyAllocs[alloc] = true
			pe.emitAlloc(alloc, b)
		}

		fmt.Fprintf(b, "    hg_panic_frame_t _hg_frame;\n")
		fmt.Fprintf(b, "    hg_panic_frame_push(&_hg_frame);\n")
		// On panic: pop frame, run defers (one may call recover()), repanic if not
		// recovered, then fall through to the recover block.
		fmt.Fprintf(b, "    if (setjmp(_hg_frame.buf) != 0) { hg_panic_frame_pop(&_hg_frame); hg_run_defers((hg_defer_t*)_hg_defer_head); hg_repanic(); goto blk%d; }\n",
			fn.Recover.Index)
	}

	pe.emitLocalDecls(fn, b)
	pe.emitRangeDecls(fn, b)

	for i, blk := range fn.Blocks {
		if i == 0 {
			// ; after label: C11 requires a statement after a label (not a declaration)
			fmt.Fprintf(b, "    goto blk0;\nblk0:;\n")
		} else {
			fmt.Fprintf(b, "blk%d:;\n", blk.Index)
		}
		pe.emitBlock(fn, blk, b)
	}

	fmt.Fprintf(b, "}\n\n")
}

// emitLocalDecls declares phi variables at the top of the function.
func (pe *pkgEmitter) emitLocalDecls(fn *ssa.Function, b *bytes.Buffer) {
	seen := map[ssa.Value]bool{}
	for _, blk := range fn.Blocks {
		for _, instr := range blk.Instrs {
			phi, ok := instr.(*ssa.Phi)
			if !ok || seen[phi] {
				continue
			}
			seen[phi] = true
			ct := pe.e.cTypeOf(phi.Type())
			fmt.Fprintf(b, "    %s %s;\n", ct, valueName(phi))
			// zero-initialize
			fmt.Fprintf(b, "    memset(&%s, 0, sizeof(%s));\n", valueName(phi), valueName(phi))
		}
	}
}

// emitNeededSliceTypes declares any slice typedefs this function needs.
func (pe *pkgEmitter) emitNeededSliceTypes(fn *ssa.Function) {
	walk := func(t types.Type) {
		sl, ok := t.Underlying().(*types.Slice)
		if !ok {
			return
		}
		elem := pe.e.cTypeInner(sl.Elem().Underlying())
		name := sliceTypeName(elem)
		if !pe.e.sliceTypes[name] {
			pe.e.sliceTypes[name] = true
			fmt.Fprint(pe.e.hdrbuf, sliceTypeDecl(elem))
		}
	}
	for _, p := range fn.Params {
		walk(p.Type())
	}
	for _, blk := range fn.Blocks {
		for _, instr := range blk.Instrs {
			if v, ok := instr.(ssa.Value); ok {
				walk(v.Type())
			}
		}
	}
}

func (pe *pkgEmitter) emitBlock(fn *ssa.Function, blk *ssa.BasicBlock, b *bytes.Buffer) {
	pe.inRecoverBlk = fn.Recover != nil && blk == fn.Recover
	for i, instr := range blk.Instrs {
		pe.emitInstr(fn, blk, instr, i == len(blk.Instrs)-1, b)
	}
	pe.inRecoverBlk = false
}

func (pe *pkgEmitter) fnHasDefers(fn *ssa.Function) bool {
	for _, blk := range fn.Blocks {
		for _, instr := range blk.Instrs {
			if _, ok := instr.(*ssa.Defer); ok {
				return true
			}
		}
	}
	return false
}

func paramName(p *ssa.Parameter, i int) string {
	n := p.Name()
	if n == "" || n == "_" {
		return fmt.Sprintf("_p%d", i)
	}
	return sanitizeName(n)
}

func valueName(v ssa.Value) string {
	n := v.Name()
	if n == "" {
		return "_anon"
	}
	return sanitizeName(n)
}

func sanitizeName(n string) string {
	// Handle method names like "(Point).Distance" → "Point_Distance"
	// and "(T).Method" patterns, plus composite type strings like "[]int" or "map[K]V".
	n = strings.TrimPrefix(n, "(")
	n = strings.NewReplacer(
		").", "_", ")", "_", "$", "_", ".", "_", "-", "_", "*", "ptr",
		"[", "sl_", "]", "", " ", "_", "{", "", "}", "",
	).Replace(n)
	switch n {
	case "default", "return", "if", "else", "for", "while", "do",
		"int", "long", "short", "char", "float", "double", "void",
		"struct", "union", "enum", "typedef", "extern", "static",
		"const", "volatile", "register", "auto", "inline", "goto",
		"break", "continue", "switch", "case", "sizeof", "bool":
		return "_" + n
	}
	return n
}

func (pe *pkgEmitter) posStr(pos token.Pos) string {
	if !pos.IsValid() {
		return "?"
	}
	p := pe.e.fset.Position(pos)
	// Escape backslashes so the path is valid in a C string literal (Windows paths).
	filename := strings.ReplaceAll(p.Filename, `\`, `\\`)
	return fmt.Sprintf("%s:%d", filename, p.Line)
}
