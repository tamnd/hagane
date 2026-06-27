#pragma once
/* hagane runtime — included by all emitted C files */
#include <stdint.h>
#include <stdbool.h>
#include <stddef.h>
#include <stdlib.h>
#include <string.h>
#include <stdio.h>
#include <math.h>
#include <setjmp.h>

/* ── string ──────────────────────────────────────────────────────────────── */
typedef struct { const char *ptr; int64_t len; } hg_string_t;

#define hg_string_lit(s) \
    ((hg_string_t){.ptr = (s), .len = (int64_t)(sizeof(s) - 1)})

#define HG_ZERO_STRING ((hg_string_t){.ptr = NULL, .len = 0})

/* ── interface fat pointer ───────────────────────────────────────────────── */
typedef struct { const void *itab; void *data; } hg_iface_t;
#define HG_ZERO_IFACE  ((hg_iface_t){.itab = NULL, .data = NULL})

/* ── function value (closure) ────────────────────────────────────────────── */
typedef struct { void *fn; void *env; } hg_func_t;

/* ── complex numbers ─────────────────────────────────────────────────────── */
typedef struct { float  re, im; } hg_complex64_t;
typedef struct { double re, im; } hg_complex128_t;

/* ── opaque runtime types ────────────────────────────────────────────────── */
typedef struct hg_map  hg_map_t;   /* completed in hagane_map.h */
typedef struct hg_chan hg_chan_t;

/* ── common slice types ──────────────────────────────────────────────────── */
typedef struct { bool      *ptr; int64_t len; int64_t cap; } hg_slice_bool_t;
typedef struct { int8_t    *ptr; int64_t len; int64_t cap; } hg_slice_int8_t;
typedef struct { int16_t   *ptr; int64_t len; int64_t cap; } hg_slice_int16_t;
typedef struct { int32_t   *ptr; int64_t len; int64_t cap; } hg_slice_int32_t;
typedef struct { int64_t   *ptr; int64_t len; int64_t cap; } hg_slice_int64_t;
typedef struct { uint8_t   *ptr; int64_t len; int64_t cap; } hg_slice_uint8_t;
typedef struct { uint16_t  *ptr; int64_t len; int64_t cap; } hg_slice_uint16_t;
typedef struct { uint32_t  *ptr; int64_t len; int64_t cap; } hg_slice_uint32_t;
typedef struct { uint64_t  *ptr; int64_t len; int64_t cap; } hg_slice_uint64_t;
typedef struct { uintptr_t *ptr; int64_t len; int64_t cap; } hg_slice_uintptr_t;
typedef struct { float     *ptr; int64_t len; int64_t cap; } hg_slice_float_t;
typedef struct { double    *ptr; int64_t len; int64_t cap; } hg_slice_double_t;
typedef struct { hg_string_t *ptr; int64_t len; int64_t cap; } hg_slice_hg_string_t_t;
typedef struct { hg_iface_t  *ptr; int64_t len; int64_t cap; } hg_slice_hg_iface_t_t;
typedef struct { void       **ptr; int64_t len; int64_t cap; } hg_slice_voidptr_t;

/* ── generic raw slice (for runtime reflection) ──────────────────────────── */
typedef struct { void *ptr; int64_t len; int64_t cap; } hg_rawslice_t;

/* ── anonymous struct placeholder ────────────────────────────────────────── */
/* Used when a struct type can't be fully resolved (skipped packages, etc.) */
typedef struct { void *_hg_placeholder; } hg_anon_struct_t;
typedef struct { hg_anon_struct_t *ptr; int64_t len; int64_t cap; } hg_slice_hg_anon_struct_t_t;

/* ── safety checks ───────────────────────────────────────────────────────── */
#define hg_bounds_check(idx, len, file, line) do { \
    if ((uint64_t)(idx) >= (uint64_t)(len)) { \
        fprintf(stderr, "%s:%d: runtime error: index out of range [%lld] with length %lld\n", \
            (file), (line), (long long)(idx), (long long)(len)); \
        abort(); \
    } \
} while(0)

#define hg_nil_check(ptr, file, line) do { \
    if ((ptr) == NULL) { \
        fprintf(stderr, "%s:%d: runtime error: invalid memory address or nil pointer dereference\n", \
            (file), (line)); \
        abort(); \
    } \
} while(0)

#define hg_divcheck(y, file, line) do { \
    if ((y) == 0) { \
        fprintf(stderr, "%s:%d: runtime error: integer divide by zero\n", (file), (line)); \
        abort(); \
    } \
} while(0)

#define hg_panic(msg, file, line) do { \
    fprintf(stderr, "%s:%d: panic: %s\n", (file), (line), (msg)); \
    abort(); \
} while(0)

/* ── zero values ─────────────────────────────────────────────────────────── */
#define HG_ZERO_SLICE(T) ((T){.ptr = NULL, .len = 0, .cap = 0})

/* ── wrapping arithmetic (preserves Go's two's-complement overflow) ───────── */
#define hg_add_i8(a,b)  ((int8_t) ((uint8_t) (a)+(uint8_t) (b)))
#define hg_add_i16(a,b) ((int16_t)((uint16_t)(a)+(uint16_t)(b)))
#define hg_add_i32(a,b) ((int32_t)((uint32_t)(a)+(uint32_t)(b)))
#define hg_add_i64(a,b) ((int64_t)((uint64_t)(a)+(uint64_t)(b)))

#define hg_sub_i8(a,b)  ((int8_t) ((uint8_t) (a)-(uint8_t) (b)))
#define hg_sub_i16(a,b) ((int16_t)((uint16_t)(a)-(uint16_t)(b)))
#define hg_sub_i32(a,b) ((int32_t)((uint32_t)(a)-(uint32_t)(b)))
#define hg_sub_i64(a,b) ((int64_t)((uint64_t)(a)-(uint64_t)(b)))

#define hg_mul_i8(a,b)  ((int8_t) ((uint8_t) (a)*(uint8_t) (b)))
#define hg_mul_i16(a,b) ((int16_t)((uint16_t)(a)*(uint16_t)(b)))
#define hg_mul_i32(a,b) ((int32_t)((uint32_t)(a)*(uint32_t)(b)))
#define hg_mul_i64(a,b) ((int64_t)((uint64_t)(a)*(uint64_t)(b)))

#define hg_neg_i8(a)  ((int8_t) (-(uint8_t) (a)))
#define hg_neg_i16(a) ((int16_t)(-(uint16_t)(a)))
#define hg_neg_i32(a) ((int32_t)(-(uint32_t)(a)))
#define hg_neg_i64(a) ((int64_t)(-(uint64_t)(a)))

/* safe shifts — mask count to avoid UB on large shifts */
#define hg_shl_i8(a,b)  ((int8_t) ((uint8_t) (a) << ((b)&7)))
#define hg_shl_i16(a,b) ((int16_t)((uint16_t)(a) << ((b)&15)))
#define hg_shl_i32(a,b) ((int32_t)((uint32_t)(a) << ((b)&31)))
#define hg_shl_i64(a,b) ((int64_t)((uint64_t)(a) << ((b)&63)))

#define hg_shr_i8(a,b)  ((int8_t) ((a) >> ((b)&7)))   /* arithmetic (signed) */
#define hg_shr_i16(a,b) ((int16_t)((a) >> ((b)&15)))
#define hg_shr_i32(a,b) ((int32_t)((a) >> ((b)&31)))
#define hg_shr_i64(a,b) ((int64_t)((a) >> ((b)&63)))

#define hg_shr_u8(a,b)  ((uint8_t) (a) >> ((b)&7))    /* logical (unsigned) */
#define hg_shr_u16(a,b) ((uint16_t)(a) >> ((b)&15))
#define hg_shr_u32(a,b) ((uint32_t)(a) >> ((b)&31))
#define hg_shr_u64(a,b) ((uint64_t)(a) >> ((b)&63))

/* ── M3 type system ──────────────────────────────────────────────────────── */

/* Type kind constants (matching reflect.Kind values) */
#define HG_KIND_BOOL      1
#define HG_KIND_INT       2
#define HG_KIND_INT8      3
#define HG_KIND_INT16     4
#define HG_KIND_INT32     5
#define HG_KIND_INT64     6
#define HG_KIND_UINT      7
#define HG_KIND_UINT8     8
#define HG_KIND_UINT16    9
#define HG_KIND_UINT32   10
#define HG_KIND_UINT64   11
#define HG_KIND_UINTPTR  12
#define HG_KIND_FLOAT32  13
#define HG_KIND_FLOAT64  14
#define HG_KIND_STRING   24
#define HG_KIND_STRUCT   25
#define HG_KIND_PTR      22
#define HG_KIND_SLICE    23
#define HG_KIND_MAP      21
#define HG_KIND_FUNC     19
#define HG_KIND_IFACE    20
#define HG_KIND_ARRAY    17

/* Type descriptor — one per concrete type in the program */
typedef struct hg_type_s {
    uint32_t    size;   /* sizeof(T) */
    uint8_t     kind;   /* HG_KIND_* */
    const char *name;   /* package-qualified name or Go type string */
    const struct hg_type_s *elem; /* for slices/arrays: element type; NULL otherwise */
} hg_type_t;

/* Interface method table — one per (interface type, concrete type) pair.
   For concrete types without any interface methods (plain boxed into any),
   methods == NULL.
   stringer is non-NULL when the concrete type has an Error() or String() method
   that returns a string; used by hg_iface_fprint/sbuf for default %v printing. */
typedef struct {
    const hg_type_t *type;    /* concrete type descriptor */
    void           **methods; /* method function pointers, in interface method order */
    hg_string_t    (*stringer)(void*); /* optional: Error()/String() → string */
} hg_iface_tab_t;

/* Primitive type descriptors (defined in hagane_rt.c) */
extern const hg_type_t hg_type_bool;
extern const hg_type_t hg_type_int8;
extern const hg_type_t hg_type_int16;
extern const hg_type_t hg_type_int32;
extern const hg_type_t hg_type_int64;
extern const hg_type_t hg_type_uint8;
extern const hg_type_t hg_type_uint16;
extern const hg_type_t hg_type_uint32;
extern const hg_type_t hg_type_uint64;
extern const hg_type_t hg_type_float32;
extern const hg_type_t hg_type_float64;
extern const hg_type_t hg_type_string;
extern const hg_type_t hg_type_uintptr;

/* Primitive itab singletons (no method table) */
extern const hg_iface_tab_t hg_itab_bool;
extern const hg_iface_tab_t hg_itab_int8;
extern const hg_iface_tab_t hg_itab_int16;
extern const hg_iface_tab_t hg_itab_int32;
extern const hg_iface_tab_t hg_itab_int64;
extern const hg_iface_tab_t hg_itab_uint8;
extern const hg_iface_tab_t hg_itab_uint16;
extern const hg_iface_tab_t hg_itab_uint32;
extern const hg_iface_tab_t hg_itab_uint64;
extern const hg_iface_tab_t hg_itab_float32;
extern const hg_iface_tab_t hg_itab_float64;
extern const hg_iface_tab_t hg_itab_string;
extern const hg_iface_tab_t hg_itab_uintptr;

/* ── math helpers ────────────────────────────────────────────────────────── */
static inline double hg_math_inf(int sign) { return sign >= 0 ? (1.0/0.0) : -(1.0/0.0); }
static inline double hg_math_nan(void)     { return 0.0/0.0; }

/* ── slices.Sort / slices.IsSorted shims ─────────────────────────────────── *
 * Called by emitted code when slices.Sort/IsSorted generic instantiations   *
 * are encountered for primitive element types.                               */
static int _hg_cmp_f64(const void *a, const void *b) {
    double x = *(const double*)a, y = *(const double*)b;
    return (x > y) - (x < y);
}
static int _hg_cmp_i64(const void *a, const void *b) {
    int64_t x = *(const int64_t*)a, y = *(const int64_t*)b;
    return (x > y) - (x < y);
}
static int _hg_cmp_str(const void *a, const void *b) {
    const hg_string_t *x = (const hg_string_t*)a, *y = (const hg_string_t*)b;
    int64_t n = x->len < y->len ? x->len : y->len;
    int r = memcmp(x->ptr, y->ptr, (size_t)n);
    if (r != 0) return r;
    return (x->len > y->len) - (x->len < y->len);
}
static inline void hg_slices_sort_f64(hg_slice_double_t s) {
    qsort(s.ptr, (size_t)s.len, sizeof(double), _hg_cmp_f64);
}
static inline void hg_slices_sort_i64(hg_slice_int64_t s) {
    qsort(s.ptr, (size_t)s.len, sizeof(int64_t), _hg_cmp_i64);
}
static inline void hg_slices_sort_str(hg_slice_hg_string_t_t s) {
    qsort(s.ptr, (size_t)s.len, sizeof(hg_string_t), _hg_cmp_str);
}
static inline bool hg_slices_issorted_f64(hg_slice_double_t s) {
    for (int64_t i = 1; i < s.len; i++) {
        if (s.ptr[i] < s.ptr[i-1]) return false;
    }
    return true;
}
static inline bool hg_slices_issorted_i64(hg_slice_int64_t s) {
    for (int64_t i = 1; i < s.len; i++) {
        if (s.ptr[i] < s.ptr[i-1]) return false;
    }
    return true;
}
static inline bool hg_slices_issorted_str(hg_slice_hg_string_t_t s) {
    for (int64_t i = 1; i < s.len; i++) {
        if (_hg_cmp_str(&s.ptr[i], &s.ptr[i-1]) < 0) return false;
    }
    return true;
}

/* ── defer runtime ───────────────────────────────────────────────────────── */
typedef struct hg_defer_s {
    struct hg_defer_s *next;
    void (*fn)(void *arg);
    void  *arg;
} hg_defer_t;

static inline void hg_run_defers(hg_defer_t *head) {
    for (hg_defer_t *d = head; d != NULL; d = d->next) {
        d->fn(d->arg);
    }
}

/* ── panic/recover frame ─────────────────────────────────────────────────── */

typedef struct hg_panic_frame_s {
    jmp_buf buf;
    struct hg_panic_frame_s *prev;
} hg_panic_frame_t;

extern hg_panic_frame_t *hg_panic_top;
extern bool              hg_panic_active;
extern hg_iface_t        hg_panic_value;

static inline void hg_panic_frame_push(hg_panic_frame_t *f) {
    f->prev = hg_panic_top;
    hg_panic_top = f;
}

static inline void hg_panic_frame_pop(hg_panic_frame_t *f) {
    hg_panic_top = f->prev;
}

/* ── runtime functions ───────────────────────────────────────────────────── */
void*        hg_alloc(size_t size);
void*        hg_realloc(void *ptr, size_t old_size, size_t new_size);
void         hg_panic_typeassert(const char *have, const char *want);
void         hg_panic_iface_nil(const char *pos);
hg_string_t  hg_string_concat(hg_string_t a, hg_string_t b);
bool         hg_string_equal(hg_string_t a, hg_string_t b);
int          hg_string_compare(hg_string_t a, hg_string_t b);
void*        hg_makeslice_raw(size_t elem_size, int64_t len, int64_t cap);
void*        hg_growslice_raw(size_t elem_size, void *old_ptr, int64_t len, int64_t *cap_out, int64_t extra);
void         hg_memmove(void *dst, const void *src, size_t n);
hg_slice_uint8_t  hg_string_to_bytes(hg_string_t s);
hg_string_t       hg_bytes_to_string(hg_slice_uint8_t b);
hg_slice_int32_t  hg_string_to_runes(hg_string_t s);
hg_string_t       hg_runes_to_string(hg_slice_int32_t sl);
void         hg_runtime_init(void);
void         hg_throw(hg_iface_t val);
hg_iface_t   hg_recover(void);
void         hg_repanic(void);

/* math/bits shim — used by sort and other packages */
static inline int64_t hg_bits_Len(uint64_t x) {
    return x == 0 ? 0 : (int64_t)(64 - __builtin_clzll((unsigned long long)x));
}
static inline int64_t hg_bits_Len8(uint8_t x)  {
    return x == 0 ? 0 : (int64_t)(8  - __builtin_clz((unsigned int)x) - 24);
}
static inline int64_t hg_bits_Len16(uint16_t x) {
    return x == 0 ? 0 : (int64_t)(16 - __builtin_clz((unsigned int)x) - 16);
}
static inline int64_t hg_bits_Len32(uint32_t x) {
    return x == 0 ? 0 : (int64_t)(32 - __builtin_clz((unsigned int)x));
}
static inline int64_t hg_bits_Len64(uint64_t x) {
    return x == 0 ? 0 : (int64_t)(64 - __builtin_clzll((unsigned long long)x));
}
static inline int64_t hg_bits_OnesCount(uint64_t x)   { return (int64_t)__builtin_popcountll(x); }
static inline int64_t hg_bits_OnesCount8(uint8_t x)   { return (int64_t)__builtin_popcount(x); }
static inline int64_t hg_bits_OnesCount16(uint16_t x) { return (int64_t)__builtin_popcount(x); }
static inline int64_t hg_bits_OnesCount32(uint32_t x) { return (int64_t)__builtin_popcount(x); }
static inline int64_t hg_bits_OnesCount64(uint64_t x) { return (int64_t)__builtin_popcountll(x); }
static inline int64_t hg_bits_TrailingZeros(uint64_t x)   { return x == 0 ? 64 : (int64_t)__builtin_ctzll(x); }
static inline int64_t hg_bits_TrailingZeros8(uint8_t x)   { return x == 0 ? 8  : (int64_t)__builtin_ctz(x); }
static inline int64_t hg_bits_TrailingZeros16(uint16_t x) { return x == 0 ? 16 : (int64_t)__builtin_ctz(x); }
static inline int64_t hg_bits_TrailingZeros32(uint32_t x) { return x == 0 ? 32 : (int64_t)__builtin_ctz(x); }
static inline int64_t hg_bits_TrailingZeros64(uint64_t x) { return x == 0 ? 64 : (int64_t)__builtin_ctzll(x); }
static inline int64_t hg_bits_LeadingZeros(uint64_t x)   { return x == 0 ? 64 : (int64_t)__builtin_clzll(x); }
static inline int64_t hg_bits_LeadingZeros8(uint8_t x)   { return x == 0 ? 8  : (int64_t)__builtin_clz(x) - 24; }
static inline int64_t hg_bits_LeadingZeros16(uint16_t x) { return x == 0 ? 16 : (int64_t)__builtin_clz(x) - 16; }
static inline int64_t hg_bits_LeadingZeros32(uint32_t x) { return x == 0 ? 32 : (int64_t)__builtin_clz(x); }
static inline int64_t hg_bits_LeadingZeros64(uint64_t x) { return x == 0 ? 64 : (int64_t)__builtin_clzll(x); }
static inline uint64_t hg_bits_RotateLeft(uint64_t x, int k) {
    unsigned s = (unsigned)k & 63;
    return (x << s) | (x >> (64 - s));
}
static inline uint32_t hg_bits_RotateLeft32(uint32_t x, int k) {
    unsigned s = (unsigned)k & 31;
    return (x << s) | (x >> (32 - s));
}
static inline void hg_bits_init(void) {}

/* errors shim */
hg_iface_t hg_errors_New(hg_string_t msg);
static inline void hg_errors_init(void) {}

/* testing shim — covers testing.T and testing.B (both map to this struct) */
typedef struct hg_testing_T {
    bool               failed;
    bool               skipped;
    bool               has_failnow;
    hg_string_t        name;
    struct hg_testing_T *parent;
    jmp_buf            failnow_jmp;
    int64_t            N; /* testing.B.N — number of benchmark iterations */
} hg_testing_T;

void hg_testing_Errorf(hg_testing_T *t, hg_string_t fmt, hg_slice_hg_iface_t_t args);
void hg_testing_Logf(hg_testing_T *t, hg_string_t fmt, hg_slice_hg_iface_t_t args);
void hg_testing_Fatal(hg_testing_T *t, hg_slice_hg_iface_t_t args);
void hg_testing_Fatalf(hg_testing_T *t, hg_string_t fmt, hg_slice_hg_iface_t_t args);
void hg_testing_Error(hg_testing_T *t, hg_slice_hg_iface_t_t args);
void hg_testing_Log(hg_testing_T *t, hg_slice_hg_iface_t_t args);
void hg_testing_Fail(hg_testing_T *t);
void hg_testing_FailNow(hg_testing_T *t);
bool hg_testing_Failed(hg_testing_T *t);
void hg_testing_Skip(hg_testing_T *t, hg_slice_hg_iface_t_t args);
void hg_testing_Skipf(hg_testing_T *t, hg_string_t fmt, hg_slice_hg_iface_t_t args);
static inline void hg_testing_Helper(hg_testing_T *t) { (void)t; }
static inline void hg_testing_Parallel(hg_testing_T *t) { (void)t; }
static inline void hg_testing_StartTimer(hg_testing_T *t) { (void)t; }
static inline void hg_testing_StopTimer(hg_testing_T *t) { (void)t; }
static inline void hg_testing_ResetTimer(hg_testing_T *t) { (void)t; }
static inline void hg_testing_ReportAllocs(hg_testing_T *t) { (void)t; }
bool hg_testing_T_Run(hg_testing_T *t, hg_string_t name, hg_func_t fn);
static inline void hg_testing_init(void) {}
static inline void hg_testing_Cleanup(hg_testing_T *t, hg_func_t fn) { (void)t; (void)fn; }

/* testing.M — only used from generated test main */
typedef struct hg_testing_M { int dummy; } hg_testing_M;
int hg_testing_M_Run(hg_testing_M *m);

/* fmt stubs (called from generated init functions) */
static inline void hg_fmt_init(void) {}

/* fmt.Print* via type-tagged interfaces */
void hg_fmt_println(hg_slice_hg_iface_t_t args);
void hg_fmt_print(hg_slice_hg_iface_t_t args);
void hg_fmt_printf(hg_string_t fmt, hg_slice_hg_iface_t_t args);
hg_string_t hg_fmt_sprintf(hg_string_t fmt, hg_slice_hg_iface_t_t args);

/* interface equality — compares by value (memcmp via type size) not by pointer identity */
static inline bool hg_iface_equal(hg_iface_t a, hg_iface_t b) {
    if (a.itab != b.itab) return false;
    if (a.itab == NULL) return true;
    if (a.data == b.data) return true;
    size_t sz = ((const hg_iface_tab_t*)a.itab)->type->size;
    if (sz == 0) return true;
    return memcmp(a.data, b.data, sz) == 0;
}

/* helper: print a Go string via printf */
static inline void hg_print_string(hg_string_t s) {
    if (s.ptr && s.len > 0) {
        fwrite(s.ptr, 1, (size_t)s.len, stdout);
    }
}

/* slices.SortFunc shim for slices of pointers (element type = any pointer) */
typedef int64_t (*_hg_ptrcmp_fn_t)(void*, void*, void*);
static hg_func_t _hg_sortfunc_ptr_cmpfn;
static int _hg_sortfunc_ptr_qcmp(const void* a, const void* b) {
    void* pa = *(void* const*)a;
    void* pb = *(void* const*)b;
    return (int)((_hg_ptrcmp_fn_t)_hg_sortfunc_ptr_cmpfn.fn)(_hg_sortfunc_ptr_cmpfn.env, pa, pb);
}
static inline void hg_slices_sortfunc_ptr(void** arr, int64_t n, hg_func_t cmp) {
    _hg_sortfunc_ptr_cmpfn = cmp;
    qsort(arr, (size_t)n, sizeof(void*), _hg_sortfunc_ptr_qcmp);
}

/* slices.Clone shims */
static inline hg_slice_double_t hg_slices_clone_f64(hg_slice_double_t s) {
    if (s.len == 0) return (hg_slice_double_t){NULL, 0, 0};
    double *p = (double*)hg_alloc((size_t)s.len * sizeof(double));
    memcpy(p, s.ptr, (size_t)s.len * sizeof(double));
    return (hg_slice_double_t){p, s.len, s.len};
}
static inline hg_slice_int64_t hg_slices_clone_i64(hg_slice_int64_t s) {
    if (s.len == 0) return (hg_slice_int64_t){NULL, 0, 0};
    int64_t *p = (int64_t*)hg_alloc((size_t)s.len * sizeof(int64_t));
    memcpy(p, s.ptr, (size_t)s.len * sizeof(int64_t));
    return (hg_slice_int64_t){p, s.len, s.len};
}
static inline hg_slice_hg_string_t_t hg_slices_clone_str(hg_slice_hg_string_t_t s) {
    if (s.len == 0) return (hg_slice_hg_string_t_t){NULL, 0, 0};
    hg_string_t *p = (hg_string_t*)hg_alloc((size_t)s.len * sizeof(hg_string_t));
    memcpy(p, s.ptr, (size_t)s.len * sizeof(hg_string_t));
    return (hg_slice_hg_string_t_t){p, s.len, s.len};
}

/* slices.EqualFunc shim for []float64 */
static inline bool hg_slices_equalfunc_f64(hg_slice_double_t a, hg_slice_double_t b, hg_func_t eq) {
    if (a.len != b.len) return false;
    for (int64_t i = 0; i < a.len; i++) {
        bool r = ((bool(*)(void*, double, double))eq.fn)(eq.env, a.ptr[i], b.ptr[i]);
        if (!r) return false;
    }
    return true;
}

/* math/rand/v2 shim — Xorshift64 for test PRNG */
static uint64_t _hg_rand_state = 1442695040888963407ULL;
static inline int64_t hg_rand_intn(int64_t n) {
    if (n <= 0) return 0;
    _hg_rand_state ^= _hg_rand_state << 13;
    _hg_rand_state ^= _hg_rand_state >> 7;
    _hg_rand_state ^= _hg_rand_state << 17;
    return (int64_t)((_hg_rand_state >> 1) % (uint64_t)n);
}
/* rand.Rand object — holds nothing useful; all calls go to package-level shim */
typedef struct hg_rand_Rand_t { uint64_t state; } hg_rand_Rand_t;
static inline hg_rand_Rand_t* hg_rand_New(void) {
    hg_rand_Rand_t *r = (hg_rand_Rand_t*)hg_alloc(sizeof(hg_rand_Rand_t));
    r->state = _hg_rand_state;
    return r;
}
static inline int64_t hg_rand_Rand_IntN(hg_rand_Rand_t *r, int64_t n) {
    (void)r;
    return hg_rand_intn(n);
}
static inline void hg_rand_init(void) {}
