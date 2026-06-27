package emit

import (
	"fmt"
	"go/types"
	"strings"
)

func (e *Emitter) cTypeInner(t types.Type) string {
	switch t := t.(type) {
	case *types.Basic:
		return basicCType(t.Kind())

	case *types.Pointer:
		// Use cTypeOf so Named types (structs) keep their names rather than
		// collapsing to hg_anon_struct_t via Underlying().
		return e.cTypeOf(t.Elem()) + "*"

	case *types.Slice:
		elem := e.cTypeInner(t.Elem().Underlying())
		name := sliceTypeName(elem)
		e.sliceTypes[name] = true
		return name

	case *types.Array:
		elem := e.cTypeInner(t.Elem().Underlying())
		name := fmt.Sprintf("hg_array_%s_%d_t", mangle(elem), t.Len())
		if !e.mapTypes[name] {
			e.mapTypes[name] = true
			if e.hdrbuf != nil {
				fmt.Fprint(e.hdrbuf, arrayTypeDecl(elem, t.Len()))
			}
		}
		return name

	case *types.Struct:
		// unnamed struct; use field-based name (rare in Go SSA at top level)
		return "hg_anon_struct_t"

	case *types.Interface:
		return "hg_iface_t"

	case *types.Map:
		return "hg_map_t*"

	case *types.Chan:
		return "hg_chan_t*"

	case *types.Signature:
		return "hg_func_t"

	case *types.Tuple:
		// multi-return: handled specially in func emission
		return "/* tuple */"

	default:
		return "void*"
	}
}

// cTypeNamed returns the C type for a named Go type (struct, alias, etc.).
func (e *Emitter) cTypeNamed(t *types.Named) string {
	pkg := t.Obj().Pkg()
	if pkg == nil {
		// built-in: error interface, etc.
		if t.Obj().Name() == "error" {
			return "hg_iface_t"
		}
		return "hg_builtin_" + t.Obj().Name() + "_t"
	}
	prefix := pkgCPrefix(pkg.Path())
	return prefix + t.Obj().Name() + "_t"
}

// cTypeOf dispatches between Named and underlying type.
func (e *Emitter) cTypeOf(t types.Type) string {
	switch t := t.(type) {
	case *types.Named:
		return e.cTypeNamed(t)
	case *types.Pointer:
		// Recurse through cTypeOf to preserve Named wrappers at every depth.
		return e.cTypeOf(t.Elem()) + "*"
	default:
		return e.cTypeInner(t.Underlying())
	}
}

func basicCType(k types.BasicKind) string {
	switch k {
	case types.Bool:
		return "bool"
	case types.Int:
		return "int64_t"
	case types.Int8:
		return "int8_t"
	case types.Int16:
		return "int16_t"
	case types.Int32:
		return "int32_t"
	case types.Int64:
		return "int64_t"
	case types.Uint:
		return "uint64_t"
	case types.Uint8:
		return "uint8_t"
	case types.Uint16:
		return "uint16_t"
	case types.Uint32:
		return "uint32_t"
	case types.Uint64:
		return "uint64_t"
	case types.Uintptr:
		return "uintptr_t"
	case types.Float32:
		return "float"
	case types.Float64:
		return "double"
	case types.Complex64:
		return "hg_complex64_t"
	case types.Complex128:
		return "hg_complex128_t"
	case types.String:
		return "hg_string_t"
	case types.UnsafePointer:
		return "void*"
	default:
		return "int64_t"
	}
}

func sliceTypeName(elemCType string) string {
	// Primitive C types already ending in _t (int8_t, uint64_t, uintptr_t…)
	// don't get another _t appended; hg_* types (hg_iface_t, hg_string_t)
	// do, because that's what the runtime header defines.
	if strings.HasSuffix(elemCType, "_t") && !strings.HasPrefix(elemCType, "hg_") {
		return "hg_slice_" + mangle(elemCType)
	}
	return "hg_slice_" + mangle(elemCType) + "_t"
}

// sliceTypeDecl emits "typedef struct { T *ptr; int64_t len; int64_t cap; } hg_slice_T_t;"
func sliceTypeDecl(elemCType string) string {
	name := sliceTypeName(elemCType)
	return fmt.Sprintf("typedef struct { %s *ptr; int64_t len; int64_t cap; } %s;\n", elemCType, name)
}

// arrayTypeDecl emits the C struct for a Go array type.
func arrayTypeDecl(elemCType string, n int64) string {
	name := fmt.Sprintf("hg_array_%s_%d_t", mangle(elemCType), n)
	return fmt.Sprintf("typedef struct { %s elems[%d]; } %s;\n", elemCType, n, name)
}

// mangle turns a C type string into a safe identifier fragment.
func mangle(s string) string {
	r := strings.NewReplacer(
		" ", "_",
		"*", "ptr",
		"[", "_",
		"]", "_",
	)
	return r.Replace(s)
}

// retStructName returns the C type name for a multi-return function.
func retStructName(prefix, funcName string) string {
	return prefix + funcName + "_ret_t"
}

// retStructDecl emits the typedef for a multi-return struct.
func retStructDecl(e *Emitter, prefix, funcName string, results *types.Tuple) string {
	typeName := retStructName(prefix, funcName)
	var sb strings.Builder
	sb.WriteString("typedef struct {\n")
	for i := 0; i < results.Len(); i++ {
		ct := e.cTypeOf(results.At(i).Type())
		fmt.Fprintf(&sb, "    %s r%d;\n", ct, i)
	}
	fmt.Fprintf(&sb, "} %s;\n", typeName)
	return sb.String()
}
