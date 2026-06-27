package emit

import (
	"fmt"
	"go/types"
	"sort"
	"strings"
)

// ifaceItabExpr returns the C expression for the itab pointer to use when boxing
// a value of concrete type concreteT into interface type ifaceT.
// For primitive types, returns the singleton itab address.
// For user-defined types, collects the pair and returns the generated itab name.
// Returns NULL only when the itab truly cannot be generated (non-transpiled packages).
func (e *Emitter) ifaceItabExpr(concreteT, ifaceT types.Type) string {
	if itab := primitiveItab(concreteT); itab != "" {
		return itab
	}
	// Only return NULL if the concrete or interface type is from a truly non-transpiled
	// package (we can't generate method bodies for it). Do NOT return NULL for cross-package
	// pairs: the itab is already emitted in the other package's header and we just reference it.
	if e.isNonTranspilablePair(concreteT, ifaceT) {
		return "NULL"
	}
	return e.registerItab(concreteT, ifaceT)
}

// isNonTranspilablePair returns true when either the concrete or interface type
// belongs to a package that is NOT being transpiled. In that case no C itab can
// be generated and the caller should emit NULL.
func (e *Emitter) isNonTranspilablePair(concreteT, ifaceT types.Type) bool {
	if named, ok := ifaceT.(*types.Named); ok {
		pkg := named.Obj().Pkg()
		if pkg != nil && !e.isTranspiled(pkg.Path()) {
			return true
		}
	}
	ct := concreteT
	if ptr, ok := ct.(*types.Pointer); ok {
		ct = ptr.Elem()
	}
	if named, ok := ct.(*types.Named); ok {
		pkg := named.Obj().Pkg()
		if pkg != nil && !e.isTranspiled(pkg.Path()) {
			return true
		}
	}
	return false
}

// typeDescPtr returns the C expression for the hg_type_t* of a type (for TypeAssert).
func (e *Emitter) typeDescPtr(t types.Type) string {
	if desc := primitiveTypeDesc(t); desc != "" {
		return desc
	}
	// Dereference pointer to get the named type (both *T and T are stored via the named hg_type_t).
	inner := t
	if ptr, ok := inner.(*types.Pointer); ok {
		inner = ptr.Elem()
	}
	if named, ok := inner.(*types.Named); ok {
		return "&" + e.userTypeName(named)
	}
	return "NULL"
}

// userTypeName returns the C name of the hg_type_t global for a named type.
func (e *Emitter) userTypeName(named *types.Named) string {
	pkg := named.Obj().Pkg()
	if pkg == nil {
		return "hg_type_" + sanitizeName(named.Obj().Name())
	}
	return "hg_type_" + pkgCPrefix(pkg.Path()) + sanitizeName(named.Obj().Name())
}

// userItabName returns the C name of the hg_iface_tab_t global for a (interface, concrete) pair.
func (e *Emitter) userItabName(concreteT, ifaceT types.Type) string {
	concreteStr := typeKey(concreteT)
	ifaceStr := typeKey(ifaceT)
	r := strings.NewReplacer("/", "_", ".", "_", "*", "ptr", "-", "_", " ", "_",
		"[", "sl_", "]", "", "{", "", "}", "", ";", "_", ",", "_")
	concreteStr = r.Replace(concreteStr)
	ifaceStr = r.Replace(ifaceStr)
	return "hg_itab_" + sanitizeName(ifaceStr) + "_" + sanitizeName(concreteStr)
}

// typeKey returns a stable string key for a type.
func typeKey(t types.Type) string {
	if named, ok := t.(*types.Named); ok {
		pkg := named.Obj().Pkg()
		if pkg == nil {
			return named.Obj().Name()
		}
		return pkg.Path() + "." + named.Obj().Name()
	}
	return t.String()
}

// ifacePair is a (interface type, concrete type) pair.
type ifacePair struct {
	concreteKey string
	ifaceKey    string
	concrete    types.Type
	iface       types.Type
}

// registerItab registers a (concrete, iface) pair and returns the C itab name.
func (e *Emitter) registerItab(concreteT, ifaceT types.Type) string {
	key := typeKey(concreteT) + " → " + typeKey(ifaceT)
	if e.ifacePairs == nil {
		e.ifacePairs = map[string]ifacePair{}
	}
	if _, ok := e.ifacePairs[key]; !ok {
		e.ifacePairs[key] = ifacePair{
			concreteKey: typeKey(concreteT),
			ifaceKey:    typeKey(ifaceT),
			concrete:    concreteT,
			iface:       ifaceT,
		}
	}
	return "&" + e.userItabName(concreteT, ifaceT)
}

// compositeTypeDescName returns the C variable name for a hg_type_t of a composite type.
// For primitive/named types, returns the existing descriptor name.
// For slices, returns "hg_type_sl_X" where X is the element type name.
func (e *Emitter) compositeTypeDescName(t types.Type) string {
	switch u := t.(type) {
	case *types.Slice:
		elemName := e.compositeTypeDescName(u.Elem())
		return "hg_type_sl_" + sanitizeName(elemName)
	case *types.Named:
		return e.userTypeName(u)
	case *types.Pointer:
		if named, ok := u.Elem().(*types.Named); ok {
			return e.userTypeName(named)
		}
	case *types.Basic:
		switch u.Kind() {
		case types.Bool:
			return "hg_type_bool"
		case types.Int8:
			return "hg_type_int8"
		case types.Int16:
			return "hg_type_int16"
		case types.Int32:
			return "hg_type_int32"
		case types.Int, types.Int64:
			return "hg_type_int64"
		case types.Uint8:
			return "hg_type_uint8"
		case types.Uint16:
			return "hg_type_uint16"
		case types.Uint32:
			return "hg_type_uint32"
		case types.Uint, types.Uint64:
			return "hg_type_uint64"
		case types.Float32:
			return "hg_type_float32"
		case types.Float64:
			return "hg_type_float64"
		case types.String:
			return "hg_type_string"
		case types.Uintptr:
			return "hg_type_uintptr"
		}
	}
	return ""
}

// emitSliceTypeDesc emits a hg_type_t global for a slice type (if not already emitted).
func (e *Emitter) emitSliceTypeDesc(sliceT *types.Slice, seenTypes map[string]bool) {
	name := e.compositeTypeDescName(sliceT)
	if name == "" || seenTypes[name] {
		return
	}
	seenTypes[name] = true
	// Recursively emit element type desc if it's also a slice.
	if elemSlice, ok := sliceT.Elem().(*types.Slice); ok {
		e.emitSliceTypeDesc(elemSlice, seenTypes)
	}
	// If the element is a named type, ensure its type descriptor is emitted too.
	if elemNamed, ok := sliceT.Elem().(*types.Named); ok {
		elemName := e.userTypeName(elemNamed)
		if !seenTypes[elemName] {
			seenTypes[elemName] = true
			e.emitUserTypeDesc(elemNamed, elemName)
		}
	}
	ct := e.cTypeOf(sliceT)
	elemName := e.compositeTypeDescName(sliceT.Elem())
	goName := sliceT.String()
	elemExpr := "NULL"
	if elemName != "" {
		elemExpr = "&" + elemName
	}
	fmt.Fprintf(e.hdrbuf, "static const hg_type_t %s = {sizeof(%s), HG_KIND_SLICE, \"%s\", %s};\n",
		name, ct, goName, elemExpr)
}

// emitIfaceDecls emits GoType globals and itab globals for all registered (iface, concrete) pairs.
// Called at the end of emitPkg, after all functions are emitted.
func (e *Emitter) emitIfaceDecls() {
	if len(e.ifacePairs) == 0 {
		return
	}

	// Sort for deterministic output
	var keys []string
	for k := range e.ifacePairs {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Emit hg_type_t globals for each unique concrete type
	seenTypes := map[string]bool{}
	for _, k := range keys {
		pair := e.ifacePairs[k]
		switch ct := pair.concrete.(type) {
		case *types.Slice:
			e.emitSliceTypeDesc(ct, seenTypes)
		default:
			// Resolve named type from both T and *T concrete types.
			concreteNamed := func(t types.Type) *types.Named {
				if ptr, ok := t.(*types.Pointer); ok {
					t = ptr.Elem()
				}
				named, _ := t.(*types.Named)
				return named
			}(pair.concrete)
			if concreteNamed != nil {
				typeName := e.userTypeName(concreteNamed)
				if !seenTypes[typeName] {
					seenTypes[typeName] = true
					e.emitUserTypeDesc(concreteNamed, typeName)
				}
			}
		}
	}

	// Emit itab globals for each pair.
	// Only emit when the concrete type's package is the current package being emitted
	// (or unknown/unnamed) — types from other packages define their itabs in their own
	// headers, which we include via #include, so redefining them here causes errors.
	for _, k := range keys {
		pair := e.ifacePairs[k]
		if skip := e.shouldSkipItab(pair); skip {
			continue
		}
		e.emitItabGlobal(pair)
	}
}

// emitUserTypeDesc emits a hg_type_t global for a named user type.
// Only emits when the type belongs to the package currently being emitted;
// types from other packages are declared in their own package header.
func (e *Emitter) emitUserTypeDesc(named *types.Named, typeName string) {
	if e.pkg != nil && named.Obj().Pkg().Path() != e.pkg.Pkg.Path() {
		// The type is from another package — its descriptor lives in that package's
		// header. Don't redefine it here to avoid duplicate-definition errors.
		return
	}
	ct := e.cTypeOf(named)
	kind := goTypeKind(named.Underlying())
	qualName := named.Obj().Pkg().Path() + "." + named.Obj().Name()
	fmt.Fprintf(e.hdrbuf, "static const hg_type_t %s = {sizeof(%s), %s, \"%s\"};\n",
		typeName, ct, kind, qualName)
}

// shouldSkipItab returns true if the itab for this pair should not be emitted in the
// current package's header. We skip when:
//   - the concrete type belongs to a different transpiled package (its itab lives there)
//   - either type belongs to a non-transpiled package (we can't generate method bodies)
func (e *Emitter) shouldSkipItab(pair ifacePair) bool {
	// Resolve the named concrete type.
	ct := pair.concrete
	if ptr, ok := ct.(*types.Pointer); ok {
		ct = ptr.Elem()
	}

	// If either the interface or concrete type is from a non-transpiled package, skip.
	if named, ok := pair.iface.(*types.Named); ok {
		pkg := named.Obj().Pkg()
		if pkg != nil && !e.isTranspiled(pkg.Path()) {
			return true
		}
	}
	if named, ok := ct.(*types.Named); ok {
		pkg := named.Obj().Pkg()
		if pkg != nil && !e.isTranspiled(pkg.Path()) {
			return true
		}
		// If the concrete type belongs to a different package than the current one,
		// its itab is already emitted in that package's header (which we #include).
		// Compare by path, not pointer: test-augmented packages (e.g. sort [sort.test])
		// share the same path as the regular package but have different *types.Package objects.
		if e.pkg != nil && pkg.Path() != e.pkg.Pkg.Path() {
			return true
		}
	}

	// Slice concrete types don't belong to a specific package — emit here.
	return false
}

// emitItabGlobal emits the method pointer array and hg_iface_tab_t for a (iface, concrete) pair.
func (e *Emitter) emitItabGlobal(pair ifacePair) {
	iface, ok := pair.iface.Underlying().(*types.Interface)
	if !ok {
		return
	}

	itabName := e.userItabName(pair.concrete, pair.iface)

	// Resolve the type descriptor expression for the concrete type.
	var named *types.Named
	isNamed := false
	switch ct := pair.concrete.(type) {
	case *types.Named:
		named = ct
		isNamed = true
	case *types.Pointer:
		if n, ok := ct.Elem().(*types.Named); ok {
			named = n
			isNamed = true
		}
	}
	typeName := "NULL"
	if isNamed {
		typeName = "&" + e.userTypeName(named)
	} else if descName := e.compositeTypeDescName(pair.concrete); descName != "" {
		typeName = "&" + descName
	}

	// Determine method order (alphabetical per spec)
	mset := types.NewMethodSet(iface)
	if mset.Len() == 0 {
		fmt.Fprintf(e.hdrbuf, "static const hg_iface_tab_t %s = {%s, NULL};\n", itabName, typeName)
		return
	}

	// For each method, generate a trampoline wrapper that takes void* self.
	// Value-receiver methods need a dereference; pointer-receiver methods just cast.
	type methodEntry struct {
		wrapperName string
	}
	var methods []methodEntry
	for i := 0; i < mset.Len(); i++ {
		sel := mset.At(i)
		methodName := sel.Obj().Name()
		ifaceMethod := sel.Obj().(*types.Func)
		ifaceSig := ifaceMethod.Type().(*types.Signature)

		concreteCName := e.concreteMethodCName(pair.concrete, methodName)
		wrapperName := itabName + "_w" + fmt.Sprintf("%d", i)

		// Determine the concrete receiver type and how to pass self.
		// For *T (pointer receiver): the interface data holds the *T directly; cast void* → T*
		// For T (value receiver): the interface data holds a *T (heap copy); deref void* → *(T*)_self
		isValueReceiver := e.isValueReceiverMethod(pair.concrete, methodName)
		_, concreteIsPtr := pair.concrete.(*types.Pointer)

		var selfArg string
		if concreteIsPtr && isNamed {
			concreteCType := e.cTypeOf(named)
			if isValueReceiver {
				// Concrete is *T, but method declared on T (value receiver).
				// The interface data holds the *T; dereference to pass T by value.
				selfArg = fmt.Sprintf("*(%s*)_self", concreteCType)
			} else {
				// Concrete is *T, method declared on *T (pointer receiver).
				// The interface data holds the *T directly.
				selfArg = fmt.Sprintf("(%s*)_self", concreteCType)
			}
		} else if isNamed {
			// data holds &value; value-receiver methods get a copy
			concreteCType := e.cTypeOf(named)
			if isValueReceiver {
				selfArg = fmt.Sprintf("*(%s*)_self", concreteCType)
			} else {
				selfArg = fmt.Sprintf("(%s*)_self", concreteCType)
			}
		} else {
			selfArg = "_self"
		}

		// Build wrapper function signature: static RetType wrapperName(void* _self, params...)
		var retCT string
		switch ifaceSig.Results().Len() {
		case 0:
			retCT = "void"
		case 1:
			retCT = e.cTypeOf(ifaceSig.Results().At(0).Type())
		default:
			retCT = retStructName(pkgCPrefix(e.pkg.Pkg.Path()), methodCBaseName2(pair.concrete, methodName))
		}

		// Build param list and call arg list
		var paramParts []string
		paramParts = append(paramParts, "void* _self")
		var callArgs []string
		callArgs = append(callArgs, selfArg)
		for j := 0; j < ifaceSig.Params().Len(); j++ {
			p := ifaceSig.Params().At(j)
			pct := e.cTypeOf(p.Type())
			paramParts = append(paramParts, fmt.Sprintf("%s _p%d", pct, j))
			callArgs = append(callArgs, fmt.Sprintf("_p%d", j))
		}

		if concreteCName != "" {
			if retCT == "void" {
				fmt.Fprintf(e.hdrbuf, "static void %s(%s) { %s(%s); }\n",
					wrapperName, strings.Join(paramParts, ", "),
					concreteCName, strings.Join(callArgs, ", "))
			} else {
				fmt.Fprintf(e.hdrbuf, "static %s %s(%s) { return %s(%s); }\n",
					retCT, wrapperName, strings.Join(paramParts, ", "),
					concreteCName, strings.Join(callArgs, ", "))
			}
		} else {
			// No concrete method found — emit NULL stub
			fmt.Fprintf(e.hdrbuf, "/* no concrete method %s for %s */\n", methodName, typeKey(pair.concrete))
			wrapperName = "NULL"
		}

		methods = append(methods, methodEntry{wrapperName: wrapperName})
	}

	// Emit method pointer array
	methodsArrayName := itabName + "_methods"
	fmt.Fprintf(e.hdrbuf, "static void* %s[] = {", methodsArrayName)
	for i, m := range methods {
		if i > 0 {
			fmt.Fprintf(e.hdrbuf, ", ")
		}
		if m.wrapperName != "NULL" {
			fmt.Fprintf(e.hdrbuf, "(void*)%s", m.wrapperName)
		} else {
			fmt.Fprintf(e.hdrbuf, "NULL")
		}
	}
	fmt.Fprintf(e.hdrbuf, "};\n")

	fmt.Fprintf(e.hdrbuf, "static const hg_iface_tab_t %s = {%s, %s};\n",
		itabName, typeName, methodsArrayName)
}

// isValueReceiverMethod returns true if methodName on concreteT has a value receiver (not pointer).
func (e *Emitter) isValueReceiverMethod(concreteT types.Type, methodName string) bool {
	t := concreteT
	if ptr, ok := t.(*types.Pointer); ok {
		t = ptr.Elem()
	}
	named, ok := t.(*types.Named)
	if !ok {
		return false
	}
	for i := 0; i < named.NumMethods(); i++ {
		m := named.Method(i)
		if m.Name() == methodName {
			sig := m.Type().(*types.Signature)
			recv := sig.Recv()
			if recv == nil {
				return false
			}
			_, isPtr := recv.Type().(*types.Pointer)
			return !isPtr
		}
	}
	return false
}

// methodCBaseName2 computes the method C base name for a (concrete type, method name) pair.
func methodCBaseName2(concreteT types.Type, methodName string) string {
	t := concreteT
	if ptr, ok := t.(*types.Pointer); ok {
		t = ptr.Elem()
	}
	if named, ok := t.(*types.Named); ok {
		return sanitizeName(named.Obj().Name()) + "_" + sanitizeName(methodName)
	}
	return sanitizeName(methodName)
}

// concreteMethodCName returns the C function name for method methodName on concrete type concreteT.
// Must match the naming used by methodCBaseName in pkg.go.
func (e *Emitter) concreteMethodCName(concreteT types.Type, methodName string) string {
	t := concreteT
	if ptr, ok := t.(*types.Pointer); ok {
		t = ptr.Elem()
	}
	named, ok := t.(*types.Named)
	if !ok {
		return ""
	}
	pkg := named.Obj().Pkg()
	if pkg == nil {
		return ""
	}
	prefix := pkgCPrefix(pkg.Path())
	// methodCBaseName for a method with value receiver: TypeName_MethodName
	return prefix + sanitizeName(named.Obj().Name()) + "_" + sanitizeName(methodName)
}

// goTypeKind returns the HG_KIND_* constant string for a type.
func goTypeKind(t types.Type) string {
	switch u := t.(type) {
	case *types.Basic:
		switch u.Kind() {
		case types.Bool:
			return "HG_KIND_BOOL"
		case types.Int8:
			return "HG_KIND_INT8"
		case types.Int16:
			return "HG_KIND_INT16"
		case types.Int32:
			return "HG_KIND_INT32"
		case types.Int, types.Int64:
			return "HG_KIND_INT64"
		case types.Uint8:
			return "HG_KIND_UINT8"
		case types.Uint16:
			return "HG_KIND_UINT16"
		case types.Uint32:
			return "HG_KIND_UINT32"
		case types.Uint, types.Uint64:
			return "HG_KIND_UINT64"
		case types.Float32:
			return "HG_KIND_FLOAT32"
		case types.Float64:
			return "HG_KIND_FLOAT64"
		case types.String:
			return "HG_KIND_STRING"
		case types.Uintptr:
			return "HG_KIND_UINTPTR"
		}
	case *types.Struct:
		return "HG_KIND_STRUCT"
	case *types.Pointer:
		return "HG_KIND_PTR"
	case *types.Slice:
		return "HG_KIND_SLICE"
	case *types.Map:
		return "HG_KIND_MAP"
	case *types.Interface:
		return "HG_KIND_IFACE"
	case *types.Signature:
		return "HG_KIND_FUNC"
	case *types.Array:
		return "HG_KIND_ARRAY"
	}
	return "HG_KIND_STRUCT"
}

// invokeRetStructName returns the C struct name for an Invoke multi-return result.
func (e *Emitter) invokeRetStructName(sig *types.Signature) string {
	// Generate a stable name based on the return types
	var parts []string
	for i := 0; i < sig.Results().Len(); i++ {
		parts = append(parts, sanitizeName(e.cTypeOf(sig.Results().At(i).Type())))
	}
	return "_hg_ret_" + strings.Join(parts, "_") + "_t"
}

// emitInvokeRetStruct emits the return struct typedef for an Invoke call if not yet emitted.
func (e *Emitter) emitInvokeRetStruct(sig *types.Signature, name string) {
	if e.nextTypes[name] {
		return
	}
	e.nextTypes[name] = true
	fmt.Fprintf(e.hdrbuf, "typedef struct { ")
	for i := 0; i < sig.Results().Len(); i++ {
		ct := e.cTypeOf(sig.Results().At(i).Type())
		fmt.Fprintf(e.hdrbuf, "%s r%d; ", ct, i)
	}
	fmt.Fprintf(e.hdrbuf, "} %s;\n", name)
}

// typeAssertTupleName returns the C struct name for a TypeAssert (T, bool) result.
func (e *Emitter) typeAssertTupleName(assertedType types.Type) string {
	ct := sanitizeName(e.cTypeOf(assertedType))
	return "_hg_ta_" + ct + "_t"
}

// emitTypeAssertTuple emits the (T, bool) tuple typedef for a comma-ok TypeAssert.
func (e *Emitter) emitTypeAssertTuple(assertedType types.Type, name string) {
	if e.nextTypes[name] {
		return
	}
	e.nextTypes[name] = true
	ct := e.cTypeOf(assertedType)
	fmt.Fprintf(e.hdrbuf, "typedef struct { %s r0; bool r1; } %s;\n", ct, name)
}
