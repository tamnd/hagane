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
	e   *Emitter
	pkg *ssa.Package
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
		fmt.Fprintf(h, "%s %s;\n", elemCT, cname)
	}

	// Collect named types in sorted order for deterministic output.
	var typeNames []string
	for name, mem := range pe.pkg.Members {
		if _, ok := mem.(*ssa.Type); ok {
			typeNames = append(typeNames, name)
		}
	}
	sort.Strings(typeNames)

	for _, name := range typeNames {
		tp := pe.pkg.Members[name].(*ssa.Type)
		named, ok := tp.Type().(*types.Named)
		if !ok {
			continue
		}
		cname := pe.e.cTypeNamed(named)
		switch st := named.Underlying().(type) {
		case *types.Struct:
			fmt.Fprintf(h, "typedef struct {\n")
			for i := 0; i < st.NumFields(); i++ {
				f := st.Field(i)
				ft := pe.e.cTypeOf(f.Type())
				fmt.Fprintf(h, "    %s %s;\n", ft, f.Name())
			}
			fmt.Fprintf(h, "} %s;\n", cname)
		default:
			// Named alias over a non-struct type (array, basic, etc.).
			// cTypeInner will register any needed array typedefs in hdrbuf.
			underlying := pe.e.cTypeInner(named.Underlying())
			fmt.Fprintf(h, "typedef %s %s;\n", underlying, cname)
		}
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
			decl := retStructDecl(pe.e, pe.prefix(), sanitizeName(fn.Name()), fn.Signature.Results())
			fmt.Fprintf(h, "%s\n", decl)
		}
	}

	// Emit function prototypes.
	for _, name := range fnNames {
		fn := pe.pkg.Members[name].(*ssa.Function)
		fmt.Fprintf(h, "%s;\n", pe.funcSignature(fn))
	}
}

func (pe *pkgEmitter) funcSignature(fn *ssa.Function) string {
	cname := pe.prefix() + sanitizeName(fn.Name())
	retType := pe.returnCType(fn)

	var params []string
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

// emitFunc emits a complete C function definition into e.buf.
func (pe *pkgEmitter) emitFunc(fn *ssa.Function) {
	b := pe.e.buf

	pe.emitNeededSliceTypes(fn)

	fmt.Fprintf(b, "%s {\n", pe.funcSignature(fn))
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
	// collect phi assignments to emit on predecessor edges
	// (we emit them inline before the branch; handled in emitInstr for If/Jump)
	for i, instr := range blk.Instrs {
		pe.emitInstr(fn, blk, instr, i == len(blk.Instrs)-1, b)
	}
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
	n = strings.NewReplacer("$", "_", ".", "_", "-", "_").Replace(n)
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
	return fmt.Sprintf("%s:%d", p.Filename, p.Line)
}
