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
		if !e.sliceTypes[name] {
			e.sliceTypes[name] = true
			if e.hdrbuf != nil {
				fmt.Fprint(e.hdrbuf, sliceTypeDecl(elem))
			}
		}
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
		return e.anonStructCType(t)

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
	// testing.T and testing.common both map to the same C shim struct.
	if pkg.Path() == "testing" {
		switch t.Obj().Name() {
		case "T", "common", "B", "F":
			return "hg_testing_T"
		case "M":
			return "hg_testing_M"
		}
	}
	// For named types from non-transpiled packages, use the appropriate placeholder:
	// - Interface types → hg_iface_t (all interfaces are the same two-word struct)
	// - Everything else → hg_anon_struct_t (opaque but complete placeholder struct)
	if !e.isTranspiled(pkg.Path()) {
		if _, isIface := t.Underlying().(*types.Interface); isIface {
			return "hg_iface_t"
		}
		return "hg_anon_struct_t"
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

// anonStructCType returns (and if necessary emits) a C typedef for an anonymous
// Go struct type. Two anonymous structs with the same layout map to the same name.
//
// Fields are emitted using cTypeInner(field.Type().Underlying()) rather than
// cTypeOf(field.Type()), so that named-type aliases (like earthMass float64)
// from packages not yet defined in the header resolve to their primitive
// equivalents (double) without creating forward-reference problems.
func (e *Emitter) anonStructCType(t *types.Struct) string {
	key := t.String()
	if name, ok := e.anonStructNames[key]; ok {
		return name
	}
	// Generate a stable name from the field names.
	var parts []string
	for i := 0; i < t.NumFields(); i++ {
		parts = append(parts, sanitizeName(t.Field(i).Name()))
	}
	base := strings.Join(parts, "_")
	if base == "" {
		base = "empty"
	}
	name := "hg_anon_" + base + "_t"
	if e.anonStructNames == nil {
		e.anonStructNames = make(map[string]string)
	}
	// If the base name is already used by a different struct layout, add a counter.
	// We detect collisions by checking whether any existing entry maps to the same C name.
	for _, usedName := range e.anonStructNames {
		if usedName == name {
			name = fmt.Sprintf("hg_anon_%d_%s_t", len(e.anonStructNames)+1, base)
			break
		}
	}
	e.anonStructNames[key] = name
	if e.hdrbuf != nil {
		// Forward declaration first (so the name is visible for pointer fields in later types).
		fmt.Fprintf(e.hdrbuf, "typedef struct %s %s;\n", name, name)
		// Compute field C types first (may write slice typedefs to hdrbuf as side effects).
		// Use cTypeInner(f.Type().Underlying()) to avoid referencing named-type aliases that
		// may not yet be defined (anonymous structs are often emitted during globals processing,
		// before the named-type-alias pass in emitHeader).
		type fieldInfo struct{ ct, nm string }
		fields := make([]fieldInfo, t.NumFields())
		for i := 0; i < t.NumFields(); i++ {
			f := t.Field(i)
			// Use Underlying() to collapse named aliases to their primitive C types.
			// cTypeInner handles Pointer, Slice, Struct, Basic — all safe here.
			ft := e.cTypeInner(f.Type().Underlying())
			fields[i] = fieldInfo{ft, f.Name()}
		}
		// Now emit the struct body (all side effects on hdrbuf already happened above).
		fmt.Fprintf(e.hdrbuf, "struct %s {\n", name)
		for _, fi := range fields {
			fmt.Fprintf(e.hdrbuf, "    %s %s;\n", fi.ct, fi.nm)
		}
		fmt.Fprintf(e.hdrbuf, "};\n")
	}
	return name
}

// sliceTypeDecl emits "typedef struct { T *ptr; int64_t len; int64_t cap; } hg_slice_T_t;"
func sliceTypeDecl(elemCType string) string {
	name := sliceTypeName(elemCType)
	g := "HG_TYPEDEF_" + strings.ToUpper(mangle(name))
	return fmt.Sprintf("#ifndef %s\n#define %s\ntypedef struct { %s *ptr; int64_t len; int64_t cap; } %s;\n#endif\n", g, g, elemCType, name)
}

// arrayTypeDecl emits the C struct for a Go array type.
func arrayTypeDecl(elemCType string, n int64) string {
	name := fmt.Sprintf("hg_array_%s_%d_t", mangle(elemCType), n)
	g := "HG_TYPEDEF_" + strings.ToUpper(mangle(name))
	return fmt.Sprintf("#ifndef %s\n#define %s\ntypedef struct { %s elems[%d]; } %s;\n#endif\n", g, g, elemCType, n, name)
}

// safeSkipCType returns a safe C type for the return value of a skipped call.
// Named types from non-transpiled packages become void* to avoid undefined references.
func (e *Emitter) safeSkipCType(t types.Type) string {
	switch t := t.(type) {
	case *types.Named:
		pkg := t.Obj().Pkg()
		if pkg != nil {
			if !e.isTranspiled(pkg.Path()) {
				return "void*"
			}
		}
	case *types.Pointer:
		inner := e.safeSkipCType(t.Elem())
		return inner + "*"
	}
	return e.cTypeOf(t)
}

// isTranspiled returns true if the given package path is being emitted as C.
// When e.transpiled is non-empty (populated by EmitAll or EmitAllTest), we
// check the map. When it's empty (e.g. EmitPkg called in isolation), we fall
// back to path analysis so that user packages and stdlib allowlist still work.
func (e *Emitter) isTranspiled(path string) bool {
	if len(e.transpiled) > 0 {
		return e.transpiled[path]
	}
	// No transpiled set: use static allowlist + user package detection.
	first := path
	if i := strings.IndexByte(first, '/'); i >= 0 {
		first = first[:i]
	}
	if strings.ContainsRune(first, '.') {
		return true // user package
	}
	return transpilableStdlib[path]
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
