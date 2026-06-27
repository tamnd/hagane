package emit

import (
	"bytes"
	"fmt"
	"go/constant"
	"go/token"
	"go/types"
	"strings"

	"golang.org/x/tools/go/ssa"

	"github.com/tamnd/hagane/frontend"
)

// emitInstr translates one SSA instruction to C.
func (pe *pkgEmitter) emitInstr(fn *ssa.Function, blk *ssa.BasicBlock, instr ssa.Instruction, isLast bool, b *bytes.Buffer) {
	switch v := instr.(type) {
	case *ssa.BinOp:
		pe.emitBinOp(v, b)

	case *ssa.UnOp:
		pe.emitUnOp(v, b)

	case *ssa.Call:
		pe.emitCall(v, b)

	case *ssa.Return:
		pe.emitReturn(fn, v, b)

	case *ssa.If:
		// emit phi assignments for each successor before branching
		pe.emitPhiAssigns(v.Cond, blk.Succs[0], blk, b)
		pe.emitPhiAssigns(v.Cond, blk.Succs[1], blk, b)
		fmt.Fprintf(b, "    if (%s) goto blk%d; else goto blk%d;\n",
			pe.emitValue(v.Cond), blk.Succs[0].Index, blk.Succs[1].Index)

	case *ssa.Jump:
		pe.emitPhiAssigns(nil, blk.Succs[0], blk, b)
		fmt.Fprintf(b, "    goto blk%d;\n", blk.Succs[0].Index)

	case *ssa.Phi:
		// phi variables are declared at function top; assignments happen on pred edges
		// nothing to emit here

	case *ssa.Alloc:
		pe.emitAlloc(v, b)

	case *ssa.Store:
		pe.emitStore(v, b)

	case *ssa.Field:
		ct := pe.e.cTypeOf(v.Type())
		xName := pe.emitValue(v.X)
		st := v.X.Type().Underlying().(*types.Struct)
		fieldName := st.Field(v.Field).Name()
		fmt.Fprintf(b, "    %s %s = %s.%s;\n", ct, valueName(v), xName, fieldName)

	case *ssa.FieldAddr:
		ct := pe.e.cTypeOf(v.Type())
		xName := pe.emitValue(v.X)
		// v.X is a pointer to struct
		var st *types.Struct
		ptr, ok := v.X.Type().Underlying().(*types.Pointer)
		if ok {
			st, _ = ptr.Elem().Underlying().(*types.Struct)
		}
		if st == nil {
			fmt.Fprintf(b, "    %s %s = NULL; /* FieldAddr: unknown struct */\n", ct, valueName(v))
			return
		}
		fieldName := st.Field(v.Field).Name()
		fmt.Fprintf(b, "    %s %s = &(%s)->%s;\n", ct, valueName(v), xName, fieldName)

	case *ssa.Index:
		pe.emitIndex(v, b)

	case *ssa.IndexAddr:
		pe.emitIndexAddr(v, b)

	case *ssa.Slice:
		pe.emitSlice(v, b)

	case *ssa.MakeSlice:
		pe.emitMakeSlice(v, b)

	case *ssa.Convert:
		fromT := v.X.Type().Underlying()
		toT := v.Type().Underlying()
		ct := pe.e.cTypeOf(v.Type())
		xName := pe.emitValue(v.X)
		// string ↔ []byte conversions
		if isString(fromT) && isByteSlice(toT) {
			fmt.Fprintf(b, "    %s %s; { hg_slice_uint8_t _bs = {.ptr=(uint8_t*)%s.ptr, .len=%s.len, .cap=%s.len}; %s=_bs; }\n",
				ct, valueName(v), xName, xName, xName, valueName(v))
			return
		}
		if isByteSlice(fromT) && isString(toT) {
			fmt.Fprintf(b, "    %s %s = (hg_string_t){.ptr=(const char*)%s.ptr, .len=%s.len};\n",
				ct, valueName(v), xName, xName)
			return
		}
		fmt.Fprintf(b, "    %s %s = (%s)(%s);\n", ct, valueName(v), ct, xName)

	case *ssa.ChangeType:
		ct := pe.e.cTypeOf(v.Type())
		fmt.Fprintf(b, "    %s %s = (%s)(%s);\n", ct, valueName(v), ct, pe.emitValue(v.X))

	case *ssa.ChangeInterface:
		ct := pe.e.cTypeOf(v.Type())
		fmt.Fprintf(b, "    %s %s = %s; /* ChangeInterface */\n", ct, valueName(v), pe.emitValue(v.X))

	case *ssa.MakeInterface:
		ct := pe.e.cTypeOf(v.Type())
		xName := pe.emitValue(v.X)
		xt := pe.e.cTypeOf(v.X.Type())
		tag := m0TypeTag(v.X.Type())
		fmt.Fprintf(b, "    %s %s; { %s *_box = (%s*)hg_alloc(sizeof(%s)); *_box = %s; %s.data = _box; %s.itab = %s; }\n",
			ct, valueName(v), xt, xt, xt, xName, valueName(v), valueName(v), tag)

	case *ssa.TypeAssert:
		// M0: panic on type mismatch (no reflect yet)
		ct := pe.e.cTypeOf(v.Type())
		fmt.Fprintf(b, "    %s %s; memset(&%s, 0, sizeof(%s)); /* TypeAssert M0 stub */\n",
			ct, valueName(v), valueName(v), valueName(v))

	case *ssa.Extract:
		// extract field from multi-return struct
		parent := pe.emitValue(v.Tuple)
		ct := pe.e.cTypeOf(v.Type())
		fmt.Fprintf(b, "    %s %s = %s.r%d;\n", ct, valueName(v), parent, v.Index)

	case *ssa.Panic:
		pos := pe.posStr(v.Pos())
		fmt.Fprintf(b, "    hg_panic(\"explicit panic\", \"%s\", 0);\n", pos)

	case *ssa.RunDefers:
		// M0 stub: defers not supported yet
		fmt.Fprintf(b, "    /* RunDefers: not supported in M0 */\n")

	case *ssa.Defer:
		fmt.Fprintf(b, "    /* Defer: not supported in M0 */\n")

	case *ssa.Go:
		fmt.Fprintf(b, "    /* Go (goroutine): not supported in M0 */\n")

	case *ssa.Send:
		fmt.Fprintf(b, "    /* Send (channel): not supported in M0 */\n")

	case *ssa.Select:
		fmt.Fprintf(b, "    /* Select: not supported in M0 */\n")

	case *ssa.MapUpdate:
		fmt.Fprintf(b, "    /* MapUpdate: not supported in M0 */\n")

	case *ssa.Lookup:
		ct := pe.e.cTypeOf(v.Type())
		fmt.Fprintf(b, "    %s %s; memset(&%s, 0, sizeof(%s)); /* Lookup M0 stub */\n",
			ct, valueName(v), valueName(v), valueName(v))

	case *ssa.MakeMap:
		ct := pe.e.cTypeOf(v.Type())
		fmt.Fprintf(b, "    %s %s = NULL; /* MakeMap M0 stub */\n", ct, valueName(v))

	case *ssa.MakeChan:
		ct := pe.e.cTypeOf(v.Type())
		fmt.Fprintf(b, "    %s %s = NULL; /* MakeChan M0 stub */\n", ct, valueName(v))

	case *ssa.MakeClosure:
		ct := pe.e.cTypeOf(v.Type())
		fmt.Fprintf(b, "    %s %s; memset(&%s, 0, sizeof(%s)); /* MakeClosure M0 stub */\n",
			ct, valueName(v), valueName(v), valueName(v))

	case *ssa.Next:
		ct := pe.e.cTypeOf(v.Type())
		fmt.Fprintf(b, "    %s %s; memset(&%s, 0, sizeof(%s)); /* Next M0 stub */\n",
			ct, valueName(v), valueName(v), valueName(v))

	case *ssa.Range:
		ct := pe.e.cTypeOf(v.Type())
		fmt.Fprintf(b, "    %s %s; memset(&%s, 0, sizeof(%s)); /* Range M0 stub */\n",
			ct, valueName(v), valueName(v), valueName(v))

	case *ssa.DebugRef:
		// nothing to emit

	default:
		fmt.Fprintf(b, "    /* unhandled: %T */\n", instr)
	}
}

// emitPhiAssigns emits the phi-variable assignments for all Phi nodes in
// successor 'succ' that come from predecessor 'pred'.
func (pe *pkgEmitter) emitPhiAssigns(cond ssa.Value, succ, pred *ssa.BasicBlock, b *bytes.Buffer) {
	for _, instr := range succ.Instrs {
		phi, ok := instr.(*ssa.Phi)
		if !ok {
			break // phis are always first
		}
		for i, edge := range phi.Edges {
			if succ.Preds[i] == pred {
				fmt.Fprintf(b, "    %s = %s;\n", valueName(phi), pe.emitValue(edge))
				break
			}
		}
	}
}

func (pe *pkgEmitter) emitBinOp(v *ssa.BinOp, b *bytes.Buffer) {
	x := pe.emitValue(v.X)
	y := pe.emitValue(v.Y)
	ct := pe.e.cTypeOf(v.Type())
	op := v.Op

	// for signed integers, use wrapping macros
	basic, isBasic := v.Type().Underlying().(*types.Basic)
	isSigned := isBasic && (basic.Info()&types.IsInteger != 0) && (basic.Info()&types.IsUnsigned == 0)
	suffix := ""
	if isBasic {
		switch basic.Kind() {
		case types.Int8:
			suffix = "i8"
		case types.Int16:
			suffix = "i16"
		case types.Int32:
			suffix = "i32"
		case types.Int, types.Int64:
			suffix = "i64"
		}
	}

	var expr string
	switch op {
	case token.ADD:
		if isSigned && suffix != "" {
			expr = fmt.Sprintf("hg_add_%s(%s, %s)", suffix, x, y)
		} else if isString(v.Type().Underlying()) {
			expr = fmt.Sprintf("hg_string_concat(%s, %s)", x, y)
		} else {
			expr = fmt.Sprintf("%s + %s", x, y)
		}
	case token.SUB:
		if isSigned && suffix != "" {
			expr = fmt.Sprintf("hg_sub_%s(%s, %s)", suffix, x, y)
		} else {
			expr = fmt.Sprintf("%s - %s", x, y)
		}
	case token.MUL:
		if isSigned && suffix != "" {
			expr = fmt.Sprintf("hg_mul_%s(%s, %s)", suffix, x, y)
		} else {
			expr = fmt.Sprintf("%s * %s", x, y)
		}
	case token.QUO:
		pos := pe.posStr(v.Pos())
		fmt.Fprintf(b, "    hg_divcheck(%s, \"%s\", 0);\n", y, pos)
		expr = fmt.Sprintf("%s / %s", x, y)
	case token.REM:
		pos := pe.posStr(v.Pos())
		fmt.Fprintf(b, "    hg_divcheck(%s, \"%s\", 0);\n", y, pos)
		expr = fmt.Sprintf("%s %% %s", x, y)
	case token.AND:
		expr = fmt.Sprintf("%s & %s", x, y)
	case token.OR:
		expr = fmt.Sprintf("%s | %s", x, y)
	case token.XOR:
		expr = fmt.Sprintf("%s ^ %s", x, y)
	case token.SHL:
		if suffix != "" {
			expr = fmt.Sprintf("hg_shl_%s(%s, %s)", suffix, x, y)
		} else {
			expr = fmt.Sprintf("%s << %s", x, y)
		}
	case token.SHR:
		uSuffix := strings.Replace(suffix, "i", "u", 1)
		if basic != nil && (basic.Info()&types.IsUnsigned != 0) && uSuffix != "" {
			expr = fmt.Sprintf("hg_shr_%s(%s, %s)", uSuffix, x, y)
		} else if suffix != "" {
			expr = fmt.Sprintf("hg_shr_%s(%s, %s)", suffix, x, y)
		} else {
			expr = fmt.Sprintf("%s >> %s", x, y)
		}
	case token.AND_NOT:
		expr = fmt.Sprintf("%s & ~%s", x, y)
	case token.EQL:
		if isString(v.X.Type().Underlying()) {
			expr = fmt.Sprintf("hg_string_equal(%s, %s)", x, y)
		} else {
			expr = fmt.Sprintf("%s == %s", x, y)
		}
	case token.NEQ:
		if isString(v.X.Type().Underlying()) {
			expr = fmt.Sprintf("!hg_string_equal(%s, %s)", x, y)
		} else {
			expr = fmt.Sprintf("%s != %s", x, y)
		}
	case token.LSS:
		expr = fmt.Sprintf("%s < %s", x, y)
	case token.LEQ:
		expr = fmt.Sprintf("%s <= %s", x, y)
	case token.GTR:
		expr = fmt.Sprintf("%s > %s", x, y)
	case token.GEQ:
		expr = fmt.Sprintf("%s >= %s", x, y)
	default:
		expr = fmt.Sprintf("/* unknown binop %s */ 0", op)
	}

	fmt.Fprintf(b, "    %s %s = %s;\n", ct, valueName(v), expr)
}

func (pe *pkgEmitter) emitUnOp(v *ssa.UnOp, b *bytes.Buffer) {
	x := pe.emitValue(v.X)
	ct := pe.e.cTypeOf(v.Type())
	var expr string
	switch v.Op {
	case token.NOT:
		expr = fmt.Sprintf("!(%s)", x)
	case token.SUB:
		// wrapping negation for signed integers
		basic, ok := v.Type().Underlying().(*types.Basic)
		if ok && basic.Info()&types.IsInteger != 0 && basic.Info()&types.IsUnsigned == 0 {
			suffix := intSuffix(basic.Kind())
			if suffix != "" {
				expr = fmt.Sprintf("hg_neg_%s(%s)", suffix, x)
			} else {
				expr = fmt.Sprintf("-(%s)", x)
			}
		} else {
			expr = fmt.Sprintf("-(%s)", x)
		}
	case token.XOR:
		expr = fmt.Sprintf("~(%s)", x) // bitwise complement
	case token.MUL: // pointer deref
		pos := pe.posStr(v.Pos())
		fmt.Fprintf(b, "    hg_nil_check(%s, \"%s\", 0);\n", x, pos)
		expr = fmt.Sprintf("*(%s)", x)
	case token.ARROW: // channel receive — stub
		fmt.Fprintf(b, "    /* channel recv not supported in M0 */\n")
		fmt.Fprintf(b, "    %s %s; memset(&%s, 0, sizeof(%s));\n", ct, valueName(v), valueName(v), valueName(v))
		return
	default:
		expr = fmt.Sprintf("/* unknown unop %s */ 0", v.Op)
	}
	fmt.Fprintf(b, "    %s %s = %s;\n", ct, valueName(v), expr)
}

func (pe *pkgEmitter) emitCall(v *ssa.Call, b *bytes.Buffer) {
	cc := v.Call

	// static call
	if !cc.IsInvoke() {
		if fn, ok := cc.Value.(*ssa.Function); ok {
			if frontend.IsFmtPrint(fn) {
				pe.emitFmtCall(fn.Name(), cc.Args, b)
				return
			}
			// direct call
			cname := pkgCPrefix(fn.Package().Pkg.Path()) + sanitizeName(fn.Name())
			args := pe.formatArgs(cc.Args)
			if v.Type() == nil || v.Type() == types.Typ[types.Invalid] {
				fmt.Fprintf(b, "    %s(%s);\n", cname, args)
				return
			}
			results := fn.Signature.Results()
			if results.Len() == 0 {
				fmt.Fprintf(b, "    %s(%s);\n", cname, args)
			} else if results.Len() == 1 {
				ct := pe.e.cTypeOf(v.Type())
				fmt.Fprintf(b, "    %s %s = %s(%s);\n", ct, valueName(v), cname, args)
			} else {
				// multi-return: store in a temp struct
				retTypeName := retStructName(pkgCPrefix(fn.Package().Pkg.Path()), sanitizeName(fn.Name()))
				fmt.Fprintf(b, "    %s %s = %s(%s);\n", retTypeName, valueName(v), cname, args)
			}
			return
		}
	}

	// fallback: indirect or invoke call
	ct := pe.e.cTypeOf(v.Type())
	fmt.Fprintf(b, "    %s %s; memset(&%s, 0, sizeof(%s)); /* indirect/invoke call M0 stub */\n",
		ct, valueName(v), valueName(v), valueName(v))
}

// emitFmtCall handles fmt.Println, fmt.Printf, fmt.Print etc. as C printf calls.
func (pe *pkgEmitter) emitFmtCall(name string, args []ssa.Value, b *bytes.Buffer) {
	switch name {
	case "Println":
		if len(args) == 0 {
			fmt.Fprintf(b, "    printf(\"\\n\");\n")
			return
		}
		// if single []interface{} arg (variadic slice), use runtime println
		if len(args) == 1 && isIfaceSlice(args[0].Type()) {
			fmt.Fprintf(b, "    hg_fmt_println(%s);\n", pe.emitValue(args[0]))
			return
		}
		// otherwise direct printf
		var parts []string
		var cArgs []string
		for _, a := range args {
			_, cFmt, cArg := fmtVerb(pe, a)
			parts = append(parts, cFmt)
			if cArg != "" {
				cArgs = append(cArgs, cArg)
			}
		}
		fmt_ := strings.Join(parts, " ") + "\\n"
		if len(cArgs) == 0 {
			fmt.Fprintf(b, "    printf(\"%s\");\n", fmt_)
		} else {
			fmt.Fprintf(b, "    printf(\"%s\", %s);\n", fmt_, strings.Join(cArgs, ", "))
		}
	case "Print":
		if len(args) == 0 {
			return
		}
		if len(args) == 1 && isIfaceSlice(args[0].Type()) {
			fmt.Fprintf(b, "    hg_fmt_print(%s);\n", pe.emitValue(args[0]))
			return
		}
		var parts []string
		var cArgs []string
		for _, a := range args {
			_, cFmt, cArg := fmtVerb(pe, a)
			parts = append(parts, cFmt)
			if cArg != "" {
				cArgs = append(cArgs, cArg)
			}
		}
		fmt_ := strings.Join(parts, "")
		if len(cArgs) == 0 {
			fmt.Fprintf(b, "    printf(\"%s\");\n", fmt_)
		} else {
			fmt.Fprintf(b, "    printf(\"%s\", %s);\n", fmt_, strings.Join(cArgs, ", "))
		}
	case "Printf":
		if len(args) == 0 {
			return
		}
		// Printf args are: format string + []interface{} variadic slice
		if len(args) >= 1 {
			fmtArg := pe.emitValue(args[0])
			if len(args) >= 2 && isIfaceSlice(args[1].Type()) {
				fmt.Fprintf(b, "    hg_fmt_printf(%s, %s);\n", fmtArg, pe.emitValue(args[1]))
			} else {
				fmt.Fprintf(b, "    hg_fmt_printf(%s, HG_ZERO_SLICE(hg_slice_hg_iface_t_t));\n", fmtArg)
			}
		}
	default:
		// Fprintln, Fprintf, Sprintf etc: stub for M0
		fmt.Fprintf(b, "    /* %s: not fully supported in M0 */\n", name)
	}
}

// fmtVerb returns (kind, c_format_spec, c_arg_expr) for a value in Println args.
func fmtVerb(pe *pkgEmitter, v ssa.Value) (string, string, string) {
	t := v.Type().Underlying()
	vName := pe.emitValue(v)
	switch {
	case isString(t):
		return "string", "%.*s", fmt.Sprintf("(int)%s.len, %s.ptr", vName, vName)
	case isBool(t):
		// print "true" / "false"
		return "bool", "%s", fmt.Sprintf("(%s ? \"true\" : \"false\")", vName)
	case isUnsigned(t):
		return "uint", "%llu", fmt.Sprintf("(unsigned long long)%s", vName)
	case isInt(t):
		return "int", "%lld", fmt.Sprintf("(long long)%s", vName)
	case isFloat(t):
		return "float", "%g", vName
	default:
		return "ptr", "%p", vName
	}
}

func (pe *pkgEmitter) emitReturn(fn *ssa.Function, v *ssa.Return, b *bytes.Buffer) {
	results := fn.Signature.Results()
	switch results.Len() {
	case 0:
		fmt.Fprintf(b, "    return;\n")
	case 1:
		fmt.Fprintf(b, "    return %s;\n", pe.emitValue(v.Results[0]))
	default:
		retType := retStructName(pe.prefix(), sanitizeName(fn.Name()))
		fmt.Fprintf(b, "    { %s _ret; ", retType)
		for i, r := range v.Results {
			fmt.Fprintf(b, "_ret.r%d = %s; ", i, pe.emitValue(r))
		}
		fmt.Fprintf(b, "return _ret; }\n")
	}
}

func (pe *pkgEmitter) emitAlloc(v *ssa.Alloc, b *bytes.Buffer) {
	ct := pe.e.cTypeOf(v.Type()) // pointer type
	elemCT := pe.e.cTypeOf(v.Type().Underlying().(*types.Pointer).Elem())
	if v.Heap {
		fmt.Fprintf(b, "    %s %s = (%s)hg_alloc(sizeof(%s));\n", ct, valueName(v), ct, elemCT)
	} else {
		// stack allocation: allocate the element and take its address
		localName := "_local_" + valueName(v)
		fmt.Fprintf(b, "    %s %s; memset(&%s, 0, sizeof(%s));\n", elemCT, localName, localName, elemCT)
		fmt.Fprintf(b, "    %s %s = &%s;\n", ct, valueName(v), localName)
	}
}

func (pe *pkgEmitter) emitStore(v *ssa.Store, b *bytes.Buffer) {
	addr := pe.emitValue(v.Addr)
	val := pe.emitValue(v.Val)
	// only nil-check if the address could genuinely be nil (not a FieldAddr result)
	_, isFieldAddr := v.Addr.(*ssa.FieldAddr)
	_, isIndexAddr := v.Addr.(*ssa.IndexAddr)
	if !isFieldAddr && !isIndexAddr {
		pos := pe.posStr(v.Pos())
		fmt.Fprintf(b, "    hg_nil_check(%s, \"%s\", 0);\n", addr, pos)
	}
	fmt.Fprintf(b, "    *(%s) = %s;\n", addr, val)
}

func (pe *pkgEmitter) emitIndex(v *ssa.Index, b *bytes.Buffer) {
	ct := pe.e.cTypeOf(v.Type())
	x := pe.emitValue(v.X)
	idx := pe.emitValue(v.Index)
	pos := pe.posStr(v.Pos())

	switch t := v.X.Type().Underlying().(type) {
	case *types.Slice:
		_ = t
		fmt.Fprintf(b, "    hg_bounds_check(%s, %s.len, \"%s\", 0);\n", idx, x, pos)
		fmt.Fprintf(b, "    %s %s = %s.ptr[%s];\n", ct, valueName(v), x, idx)
	case *types.Array:
		fmt.Fprintf(b, "    hg_bounds_check(%s, %d, \"%s\", 0);\n", idx, t.Len(), pos)
		fmt.Fprintf(b, "    %s %s = %s.elems[%s];\n", ct, valueName(v), x, idx)
	case *types.Basic: // string index
		fmt.Fprintf(b, "    hg_bounds_check(%s, %s.len, \"%s\", 0);\n", idx, x, pos)
		fmt.Fprintf(b, "    %s %s = (uint8_t)%s.ptr[%s];\n", ct, valueName(v), x, idx)
	default:
		fmt.Fprintf(b, "    %s %s = 0; /* Index: unknown type %T */\n", ct, valueName(v), t)
	}
}

func (pe *pkgEmitter) emitIndexAddr(v *ssa.IndexAddr, b *bytes.Buffer) {
	ct := pe.e.cTypeOf(v.Type())
	x := pe.emitValue(v.X)
	idx := pe.emitValue(v.Index)
	pos := pe.posStr(v.Pos())

	underlying := v.X.Type().Underlying()
	// pointer-to-array: &(*ptr)[i]
	if ptr, ok := underlying.(*types.Pointer); ok {
		if arr, ok2 := ptr.Elem().Underlying().(*types.Array); ok2 {
			fmt.Fprintf(b, "    hg_bounds_check(%s, %d, \"%s\", 0);\n", idx, arr.Len(), pos)
			fmt.Fprintf(b, "    %s %s = &(*%s).elems[%s];\n", ct, valueName(v), x, idx)
			return
		}
	}
	switch t := underlying.(type) {
	case *types.Slice:
		_ = t
		fmt.Fprintf(b, "    hg_bounds_check(%s, %s.len, \"%s\", 0);\n", idx, x, pos)
		fmt.Fprintf(b, "    %s %s = &%s.ptr[%s];\n", ct, valueName(v), x, idx)
	case *types.Array:
		fmt.Fprintf(b, "    hg_bounds_check(%s, %d, \"%s\", 0);\n", idx, t.Len(), pos)
		fmt.Fprintf(b, "    %s %s = &%s.elems[%s];\n", ct, valueName(v), x, idx)
	default:
		fmt.Fprintf(b, "    %s %s = NULL; /* IndexAddr: unknown type %T */\n", ct, valueName(v), t)
	}
}

func (pe *pkgEmitter) emitSlice(v *ssa.Slice, b *bytes.Buffer) {
	ct := pe.e.cTypeOf(v.Type())
	x := pe.emitValue(v.X)

	low := "0"
	if v.Low != nil {
		low = pe.emitValue(v.Low)
	}
	high := ""
	if v.High != nil {
		high = pe.emitValue(v.High)
	}

	underlying := v.X.Type().Underlying()
	// pointer-to-array: &(*ptr)[low:high]
	if ptr, ok := underlying.(*types.Pointer); ok {
		if arr, ok2 := ptr.Elem().Underlying().(*types.Array); ok2 {
			if high == "" {
				high = fmt.Sprintf("%d", arr.Len())
			}
			fmt.Fprintf(b, "    %s %s = {.ptr = (*%s).elems+(%s), .len = (%s)-(%s), .cap = %d-(%s)};\n",
				ct, valueName(v), x, low, high, low, arr.Len(), low)
			return
		}
	}
	switch t := underlying.(type) {
	case *types.Slice:
		_ = t
		if high == "" {
			high = x + ".len"
		}
		fmt.Fprintf(b, "    %s %s = {.ptr = %s.ptr+(%s), .len = (%s)-(%s), .cap = %s.cap-(%s)};\n",
			ct, valueName(v), x, low, high, low, x, low)
	case *types.Array:
		if high == "" {
			high = fmt.Sprintf("%d", t.Len())
		}
		fmt.Fprintf(b, "    %s %s = {.ptr = %s.elems+(%s), .len = (%s)-(%s), .cap = %d-(%s)};\n",
			ct, valueName(v), x, low, high, low, t.Len(), low)
	case *types.Basic: // string slice
		if high == "" {
			high = x + ".len"
		}
		fmt.Fprintf(b, "    %s %s = {.ptr = (char*)%s.ptr+(%s), .len = (%s)-(%s)};\n",
			ct, valueName(v), x, low, high, low)
	default:
		fmt.Fprintf(b, "    %s %s; memset(&%s, 0, sizeof(%s)); /* Slice: unknown %T */\n",
			ct, valueName(v), valueName(v), valueName(v), t)
	}
}

func (pe *pkgEmitter) emitMakeSlice(v *ssa.MakeSlice, b *bytes.Buffer) {
	ct := pe.e.cTypeOf(v.Type())
	sl := v.Type().Underlying().(*types.Slice)
	elemCT := pe.e.cTypeOf(sl.Elem())
	lenV := pe.emitValue(v.Len)
	capV := pe.emitValue(v.Cap)
	fmt.Fprintf(b, "    %s %s = {.ptr = (%s*)hg_makeslice_raw(sizeof(%s), %s, %s), .len = %s, .cap = %s};\n",
		ct, valueName(v), elemCT, elemCT, lenV, capV, lenV, capV)
}

// emitValue returns the C expression string for an SSA value.
func (pe *pkgEmitter) emitValue(v ssa.Value) string {
	switch v := v.(type) {
	case *ssa.Const:
		return pe.emitConst(v)
	case *ssa.Global:
		// *ssa.Global has type *T; it represents the address of the package-level variable
		return "&" + pkgCPrefix(v.Package().Pkg.Path()) + sanitizeName(v.Name())
	case *ssa.FreeVar:
		return "_env->" + sanitizeName(v.Name())
	case *ssa.Parameter:
		return paramName(v, 0) // best effort; index not available here
	default:
		return valueName(v)
	}
}

func (pe *pkgEmitter) emitConst(c *ssa.Const) string {
	t := c.Type().Underlying()
	switch {
	case c.Value == nil:
		return pe.zeroFor(t)
	case isString(t):
		s := constant.StringVal(c.Value)
		return fmt.Sprintf("hg_string_lit(\"%s\")", escapeCStr(s))
	case isBool(t):
		if constant.BoolVal(c.Value) {
			return "true"
		}
		return "false"
	case isInt(t) || isUnsigned(t):
		ct := pe.e.cTypeOf(c.Type())
		return fmt.Sprintf("((%s)%s)", ct, c.Value.String())
	case isFloat(t):
		return c.Value.String()
	default:
		return "0"
	}
}

func (pe *pkgEmitter) zeroFor(t types.Type) string {
	switch t := t.(type) {
	case *types.Basic:
		switch {
		case isString(t):
			return "HG_ZERO_STRING"
		case isBool(t):
			return "false"
		case isInt(t) || isUnsigned(t):
			return "0"
		case isFloat(t):
			return "0.0"
		}
	case *types.Pointer, *types.Map, *types.Chan:
		return "NULL"
	case *types.Interface:
		return "HG_ZERO_IFACE"
	}
	return "{0}"
}

func (pe *pkgEmitter) formatArgs(args []ssa.Value) string {
	var parts []string
	for _, a := range args {
		parts = append(parts, pe.emitValue(a))
	}
	return strings.Join(parts, ", ")
}

// m0TypeTag returns the HG_TYPE_* constant for M0 interface boxing.
func m0TypeTag(t types.Type) string {
	b, ok := t.Underlying().(*types.Basic)
	if !ok {
		return "HG_TYPE_UNKNOWN"
	}
	switch b.Kind() {
	case types.Bool:
		return "HG_TYPE_BOOL"
	case types.Int8:
		return "HG_TYPE_INT8"
	case types.Int16:
		return "HG_TYPE_INT16"
	case types.Int32:
		return "HG_TYPE_INT32"
	case types.Int, types.Int64:
		return "HG_TYPE_INT64"
	case types.Uint8:
		return "HG_TYPE_UINT8"
	case types.Uint16:
		return "HG_TYPE_UINT16"
	case types.Uint32:
		return "HG_TYPE_UINT32"
	case types.Uint, types.Uint64:
		return "HG_TYPE_UINT64"
	case types.Float32:
		return "HG_TYPE_FLOAT32"
	case types.Float64:
		return "HG_TYPE_FLOAT64"
	case types.String:
		return "HG_TYPE_STRING"
	case types.Uintptr:
		return "HG_TYPE_UINTPTR"
	}
	return "HG_TYPE_UNKNOWN"
}

// ── type predicates ───────────────────────────────────────────────────────────

func isString(t types.Type) bool {
	b, ok := t.(*types.Basic)
	return ok && b.Kind() == types.String
}

func isBool(t types.Type) bool {
	b, ok := t.(*types.Basic)
	return ok && b.Kind() == types.Bool
}

func isInt(t types.Type) bool {
	b, ok := t.(*types.Basic)
	return ok && b.Info()&types.IsInteger != 0 && b.Info()&types.IsUnsigned == 0
}

func isUnsigned(t types.Type) bool {
	b, ok := t.(*types.Basic)
	return ok && b.Info()&types.IsUnsigned != 0
}

func isFloat(t types.Type) bool {
	b, ok := t.(*types.Basic)
	return ok && b.Info()&types.IsFloat != 0
}

func isIfaceSlice(t types.Type) bool {
	sl, ok := t.Underlying().(*types.Slice)
	if !ok {
		return false
	}
	_, ok = sl.Elem().Underlying().(*types.Interface)
	return ok
}

func isByteSlice(t types.Type) bool {
	sl, ok := t.(*types.Slice)
	if !ok {
		return false
	}
	b, ok := sl.Elem().Underlying().(*types.Basic)
	return ok && b.Kind() == types.Uint8
}

func intSuffix(k types.BasicKind) string {
	switch k {
	case types.Int8:
		return "i8"
	case types.Int16:
		return "i16"
	case types.Int32:
		return "i32"
	case types.Int, types.Int64:
		return "i64"
	}
	return ""
}

// ── string utilities ──────────────────────────────────────────────────────────

func escapeCStr(s string) string {
	var sb strings.Builder
	for _, r := range s {
		switch r {
		case '"':
			sb.WriteString(`\"`)
		case '\\':
			sb.WriteString(`\\`)
		case '\n':
			sb.WriteString(`\n`)
		case '\t':
			sb.WriteString(`\t`)
		case '\r':
			sb.WriteString(`\r`)
		default:
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

func extractStringConst(v ssa.Value) string {
	c, ok := v.(*ssa.Const)
	if !ok || c.Value == nil {
		return ""
	}
	if isString(c.Type().Underlying()) {
		return constant.StringVal(c.Value)
	}
	return ""
}

// goFmtToCFmt converts a Go format string to a C printf format string.
// This is a best-effort M0 conversion.
func goFmtToCFmt(s string) string {
	// In Go, %v on a number is the same as %d/%g; we leave it for now.
	// Replace %v with %s for strings (handled at call site via args)
	return s
}
