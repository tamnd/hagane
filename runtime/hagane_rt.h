#pragma once
#include <stdint.h>
#include <stdbool.h>
#include <stddef.h>
#include <stdlib.h>
#include <string.h>
#include <stdio.h>
#include <stdarg.h>

/* ---------------------------------------------------------------------------
 * Core types
 * ------------------------------------------------------------------------- */

/* hg_string_t: immutable UTF-8 string (pointer + length, no NUL required) */
typedef struct { const char *ptr; int64_t len; } hg_string_t;

/* hg_iface_t: Go interface fat pointer (itab + data word) */
typedef struct { const void *itab; void *data; } hg_iface_t;

/* complex number types matching Go complex64/complex128 */
typedef struct { float  re, im; } hg_complex64_t;
typedef struct { double re, im; } hg_complex128_t;

/* ---------------------------------------------------------------------------
 * Per-type slice structs
 * ------------------------------------------------------------------------- */

typedef struct { int8_t    *ptr; int64_t len; int64_t cap; } hg_slice_int8_t;
typedef struct { int16_t   *ptr; int64_t len; int64_t cap; } hg_slice_int16_t;
typedef struct { int32_t   *ptr; int64_t len; int64_t cap; } hg_slice_int32_t;
typedef struct { int64_t   *ptr; int64_t len; int64_t cap; } hg_slice_int64_t;
typedef struct { uint8_t   *ptr; int64_t len; int64_t cap; } hg_slice_uint8_t;
typedef struct { uint16_t  *ptr; int64_t len; int64_t cap; } hg_slice_uint16_t;
typedef struct { uint32_t  *ptr; int64_t len; int64_t cap; } hg_slice_uint32_t;
typedef struct { uint64_t  *ptr; int64_t len; int64_t cap; } hg_slice_uint64_t;
typedef struct { float     *ptr; int64_t len; int64_t cap; } hg_slice_float_t;
typedef struct { double    *ptr; int64_t len; int64_t cap; } hg_slice_double_t;
typedef struct { bool      *ptr; int64_t len; int64_t cap; } hg_slice_bool_t;
typedef struct { char      *ptr; int64_t len; int64_t cap; } hg_slice_char_t;
typedef struct { void     **ptr; int64_t len; int64_t cap; } hg_slice_ptr_t;
typedef struct { hg_string_t *ptr; int64_t len; int64_t cap; } hg_slice_string_t;
typedef struct { hg_iface_t  *ptr; int64_t len; int64_t cap; } hg_slice_iface_t;

/* Generic byte slice alias (Go []byte = []uint8) */
typedef hg_slice_uint8_t hg_bytes_t;

/* ---------------------------------------------------------------------------
 * Sentinel zero values
 * ------------------------------------------------------------------------- */

#define HG_ZERO_STRING ((hg_string_t){NULL, 0})
#define HG_ZERO_IFACE  ((hg_iface_t){NULL, NULL})

/* ---------------------------------------------------------------------------
 * String literal helper (compile-time length from string constant) */
#define hg_string_lit(s) ((hg_string_t){.ptr = (s), .len = (int64_t)(sizeof(s) - 1)})

/* ---------------------------------------------------------------------------
 * Panic and safety checks
 * ------------------------------------------------------------------------- */

/* hg_panic: print message with file/line then abort */
#define hg_panic(msg, file, line) \
    do { \
        fprintf(stderr, "%s:%d: panic: %s\n", (file), (line), (msg)); \
        abort(); \
    } while (0)

/* hg_bounds_check: index must be in [0, len) */
#define hg_bounds_check(idx, len, file, line) \
    do { \
        int64_t _i = (int64_t)(idx); \
        int64_t _n = (int64_t)(len); \
        if (_i < 0 || _i >= _n) { \
            fprintf(stderr, "%s:%d: panic: runtime error: index out of range [%lld] with length %lld\n", \
                    (file), (line), (long long)_i, (long long)_n); \
            abort(); \
        } \
    } while (0)

/* hg_slice_bounds_check: slice expression a[lo:hi] where hi <= cap */
#define hg_slice_bounds_check(lo, hi, cap, file, line) \
    do { \
        int64_t _lo  = (int64_t)(lo); \
        int64_t _hi  = (int64_t)(hi); \
        int64_t _cap = (int64_t)(cap); \
        if (_lo < 0 || _hi < _lo || _hi > _cap) { \
            fprintf(stderr, "%s:%d: panic: runtime error: slice bounds out of range [%lld:%lld] with capacity %lld\n", \
                    (file), (line), (long long)_lo, (long long)_hi, (long long)_cap); \
            abort(); \
        } \
    } while (0)

/* hg_nil_check: pointer must be non-NULL (nil dereference guard) */
#define hg_nil_check(ptr, file, line) \
    do { \
        if ((ptr) == NULL) { \
            fprintf(stderr, "%s:%d: panic: runtime error: invalid memory address or nil pointer dereference\n", \
                    (file), (line)); \
            abort(); \
        } \
    } while (0)

/* hg_divcheck: integer divisor must be non-zero */
#define hg_divcheck(y, file, line) \
    do { \
        if ((y) == 0) { \
            fprintf(stderr, "%s:%d: panic: runtime error: integer divide by zero\n", \
                    (file), (line)); \
            abort(); \
        } \
    } while (0)

/* hg_negshift_check: shift count must not be negative (signed shift amount) */
#define hg_negshift_check(s, file, line) \
    do { \
        if ((int64_t)(s) < 0) { \
            fprintf(stderr, "%s:%d: panic: runtime error: negative shift amount\n", \
                    (file), (line)); \
            abort(); \
        } \
    } while (0)

/* ---------------------------------------------------------------------------
 * Wrapping arithmetic (all integer widths)
 * Go integer arithmetic wraps on overflow; use unsigned intermediaries.
 * ------------------------------------------------------------------------- */

/* --- int8 --- */
#define hg_add_i8(a,b)  ((int8_t)((uint8_t)(a)  + (uint8_t)(b)))
#define hg_sub_i8(a,b)  ((int8_t)((uint8_t)(a)  - (uint8_t)(b)))
#define hg_mul_i8(a,b)  ((int8_t)((uint8_t)(a)  * (uint8_t)(b)))
#define hg_neg_i8(a)    ((int8_t)(-(uint8_t)(a)))

/* --- int16 --- */
#define hg_add_i16(a,b) ((int16_t)((uint16_t)(a) + (uint16_t)(b)))
#define hg_sub_i16(a,b) ((int16_t)((uint16_t)(a) - (uint16_t)(b)))
#define hg_mul_i16(a,b) ((int16_t)((uint16_t)(a) * (uint16_t)(b)))
#define hg_neg_i16(a)   ((int16_t)(-(uint16_t)(a)))

/* --- int32 --- */
#define hg_add_i32(a,b) ((int32_t)((uint32_t)(a) + (uint32_t)(b)))
#define hg_sub_i32(a,b) ((int32_t)((uint32_t)(a) - (uint32_t)(b)))
#define hg_mul_i32(a,b) ((int32_t)((uint32_t)(a) * (uint32_t)(b)))
#define hg_neg_i32(a)   ((int32_t)(-(uint32_t)(a)))

/* --- int64 --- */
#define hg_add_i64(a,b) ((int64_t)((uint64_t)(a) + (uint64_t)(b)))
#define hg_sub_i64(a,b) ((int64_t)((uint64_t)(a) - (uint64_t)(b)))
#define hg_mul_i64(a,b) ((int64_t)((uint64_t)(a) * (uint64_t)(b)))
#define hg_neg_i64(a)   ((int64_t)(-(uint64_t)(a)))

/* --- uint8 --- */
#define hg_add_u8(a,b)  ((uint8_t)((a)  + (b)))
#define hg_sub_u8(a,b)  ((uint8_t)((a)  - (b)))
#define hg_mul_u8(a,b)  ((uint8_t)((a)  * (b)))
#define hg_neg_u8(a)    ((uint8_t)(-(a)))

/* --- uint16 --- */
#define hg_add_u16(a,b) ((uint16_t)((a) + (b)))
#define hg_sub_u16(a,b) ((uint16_t)((a) - (b)))
#define hg_mul_u16(a,b) ((uint16_t)((a) * (b)))
#define hg_neg_u16(a)   ((uint16_t)(-(a)))

/* --- uint32 --- */
#define hg_add_u32(a,b) ((uint32_t)((a) + (b)))
#define hg_sub_u32(a,b) ((uint32_t)((a) - (b)))
#define hg_mul_u32(a,b) ((uint32_t)((a) * (b)))
#define hg_neg_u32(a)   ((uint32_t)(-(a)))

/* --- uint64 --- */
#define hg_add_u64(a,b) ((uint64_t)((a) + (b)))
#define hg_sub_u64(a,b) ((uint64_t)((a) - (b)))
#define hg_mul_u64(a,b) ((uint64_t)((a) * (b)))
#define hg_neg_u64(a)   ((uint64_t)(-(a)))

/* ---------------------------------------------------------------------------
 * Safe shift macros (mask shift count to avoid UB)
 * Signed right shifts are arithmetic (implementation-defined in C; true for
 * every compiler hagane targets: GCC/Clang on x86-64 and arm64).
 * ------------------------------------------------------------------------- */

/* int8 shifts */
#define hg_shl_i8(a,b)  ((int8_t)((uint8_t)(a)  << ((b) & 7)))
#define hg_shr_i8(a,b)  ((int8_t)((a)            >> ((b) & 7)))
#define hg_shl_u8(a,b)  ((uint8_t)((a)            << ((b) & 7)))
#define hg_shr_u8(a,b)  ((uint8_t)((a)            >> ((b) & 7)))

/* int16 shifts */
#define hg_shl_i16(a,b) ((int16_t)((uint16_t)(a) << ((b) & 15)))
#define hg_shr_i16(a,b) ((int16_t)((a)            >> ((b) & 15)))
#define hg_shl_u16(a,b) ((uint16_t)((a)           << ((b) & 15)))
#define hg_shr_u16(a,b) ((uint16_t)((a)           >> ((b) & 15)))

/* int32 shifts */
#define hg_shl_i32(a,b) ((int32_t)((uint32_t)(a) << ((b) & 31)))
#define hg_shr_i32(a,b) ((int32_t)((a)            >> ((b) & 31)))
#define hg_shl_u32(a,b) ((uint32_t)((a)           << ((b) & 31)))
#define hg_shr_u32(a,b) ((uint32_t)((a)           >> ((b) & 31)))

/* int64 shifts */
#define hg_shl_i64(a,b) ((int64_t)((uint64_t)(a) << ((b) & 63)))
#define hg_shr_i64(a,b) ((int64_t)((a)            >> ((b) & 63)))
#define hg_shl_u64(a,b) ((uint64_t)((a)           << ((b) & 63)))
#define hg_shr_u64(a,b) ((uint64_t)((a)           >> ((b) & 63)))

/* ---------------------------------------------------------------------------
 * Allocation helpers
 * hg_alloc: zeroed allocation; aborts on OOM (Go semantics: no NULL return).
 * hg_realloc: grow/shrink; old_size used for zeroing new tail on growth.
 * ------------------------------------------------------------------------- */

static inline void *hg_alloc(size_t size) {
    if (size == 0) size = 1;
    void *p = calloc(1, size);
    if (p == NULL) {
        fprintf(stderr, "hagane: out of memory (alloc %zu bytes)\n", size);
        abort();
    }
    return p;
}

static inline void *hg_realloc(void *ptr, size_t old_size, size_t new_size) {
    if (new_size == 0) new_size = 1;
    void *p = realloc(ptr, new_size);
    if (p == NULL) {
        fprintf(stderr, "hagane: out of memory (realloc %zu -> %zu bytes)\n", old_size, new_size);
        abort();
    }
    /* zero-fill newly added tail */
    if (new_size > old_size) {
        memset((char *)p + old_size, 0, new_size - old_size);
    }
    return p;
}

/* ---------------------------------------------------------------------------
 * String operations
 * ------------------------------------------------------------------------- */

static inline bool hg_string_equal(hg_string_t a, hg_string_t b) {
    if (a.len != b.len) return false;
    if (a.ptr == b.ptr) return true;
    if (a.len == 0)     return true;
    return memcmp(a.ptr, b.ptr, (size_t)a.len) == 0;
}

static inline int hg_string_compare(hg_string_t a, hg_string_t b) {
    size_t min = (size_t)(a.len < b.len ? a.len : b.len);
    int r = (min > 0) ? memcmp(a.ptr, b.ptr, min) : 0;
    if (r != 0) return r;
    if (a.len < b.len) return -1;
    if (a.len > b.len) return  1;
    return 0;
}

/* hg_string_concat: allocate a new string holding a+b */
static inline hg_string_t hg_string_concat(hg_string_t a, hg_string_t b) {
    int64_t total = a.len + b.len;
    if (total == 0) return HG_ZERO_STRING;
    char *buf = (char *)hg_alloc((size_t)total + 1);
    if (a.len > 0) memcpy(buf,          a.ptr, (size_t)a.len);
    if (b.len > 0) memcpy(buf + a.len,  b.ptr, (size_t)b.len);
    buf[total] = '\0';
    return (hg_string_t){.ptr = buf, .len = total};
}

/* hg_string_index: single byte at index i (bounds-checked) */
static inline uint8_t hg_string_index(hg_string_t s, int64_t i,
                                       const char *file, int line) {
    hg_bounds_check(i, s.len, file, line);
    return (uint8_t)s.ptr[i];
}

/* hg_string_slice: s[lo:hi] */
static inline hg_string_t hg_string_slice(hg_string_t s, int64_t lo, int64_t hi,
                                           const char *file, int line) {
    hg_slice_bounds_check(lo, hi, s.len, file, line);
    return (hg_string_t){.ptr = s.ptr + lo, .len = hi - lo};
}

/* hg_string_from_bytes: convert []uint8 to string (copies) */
static inline hg_string_t hg_string_from_bytes(const uint8_t *data, int64_t len) {
    if (len == 0) return HG_ZERO_STRING;
    char *buf = (char *)hg_alloc((size_t)len + 1);
    memcpy(buf, data, (size_t)len);
    buf[len] = '\0';
    return (hg_string_t){.ptr = buf, .len = len};
}

/* hg_string_to_bytes: string to newly-allocated []uint8 backing array */
static inline uint8_t *hg_string_to_bytes(hg_string_t s) {
    if (s.len == 0) return (uint8_t *)hg_alloc(1);
    uint8_t *buf = (uint8_t *)hg_alloc((size_t)s.len);
    memcpy(buf, s.ptr, (size_t)s.len);
    return buf;
}

/* ---------------------------------------------------------------------------
 * Raw slice helpers
 * The typed slice structs carry ptr/len/cap. The raw helpers work in bytes
 * and are called by emitted code that knows elem_size at compile time.
 * ------------------------------------------------------------------------- */

/* hg_makeslice_raw: allocate a zeroed backing array; returns the data pointer */
static inline void *hg_makeslice_raw(size_t elem_size, int64_t len, int64_t cap) {
    if (len < 0 || cap < len) {
        fprintf(stderr, "hagane: makeslice: len=%lld cap=%lld\n",
                (long long)len, (long long)cap);
        abort();
    }
    if (cap == 0) return NULL;
    return hg_alloc(elem_size * (size_t)cap);
}

/* hg_growslice_raw: grow backing array so that it can hold (len+extra) elements.
 * Returns new data pointer; writes new capacity to *cap_out.
 * Growth policy: double until 1024 elems, then 25% steps (matches Go runtime). */
static inline void *hg_growslice_raw(size_t elem_size, void *old,
                                      int64_t len, int64_t *cap_out, int64_t extra) {
    int64_t old_cap = *cap_out;
    int64_t need    = len + extra;
    int64_t new_cap = old_cap;
    if (new_cap == 0) new_cap = 1;
    while (new_cap < need) {
        if (new_cap < 1024) {
            new_cap *= 2;
        } else {
            new_cap += new_cap / 4;
        }
    }
    void *p = hg_realloc(old,
                         elem_size * (size_t)old_cap,
                         elem_size * (size_t)new_cap);
    *cap_out = new_cap;
    return p;
}

/* hg_appendslice_raw: append `count` elements from `elems` into a raw slice.
 * Caller passes current ptr/len/cap by pointer and gets updated values. */
static inline void hg_appendslice_raw(size_t elem_size,
                                       void **ptr_out, int64_t *len_out, int64_t *cap_out,
                                       const void *elems, int64_t count) {
    int64_t old_len = *len_out;
    int64_t old_cap = *cap_out;
    if (old_len + count > old_cap) {
        *ptr_out = hg_growslice_raw(elem_size, *ptr_out, old_len, cap_out, count);
    }
    memcpy((char *)*ptr_out + elem_size * (size_t)old_len,
           elems, elem_size * (size_t)count);
    *len_out = old_len + count;
}

/* hg_copyslice_raw: builtin copy(dst, src) — min(dst.len, src.len) elements */
static inline int64_t hg_copyslice_raw(size_t elem_size,
                                        void *dst, int64_t dst_len,
                                        const void *src, int64_t src_len) {
    int64_t n = dst_len < src_len ? dst_len : src_len;
    if (n > 0) memmove(dst, src, elem_size * (size_t)n);
    return n;
}

/* ---------------------------------------------------------------------------
 * memmove wrapper (typed alias used by emitted code)
 * ------------------------------------------------------------------------- */

static inline void hg_memmove(void *dst, const void *src, size_t n) {
    memmove(dst, src, n);
}

/* ---------------------------------------------------------------------------
 * Interface helpers
 * ------------------------------------------------------------------------- */

/* hg_iface_nil: true when both itab and data are NULL (Go nil interface) */
static inline bool hg_iface_nil(hg_iface_t i) {
    return i.itab == NULL;
}

/* hg_iface_eq: Go interface equality (both nil, or same itab+data) */
static inline bool hg_iface_eq(hg_iface_t a, hg_iface_t b) {
    return a.itab == b.itab && a.data == b.data;
}

/* ---------------------------------------------------------------------------
 * fmt helpers (minimal, used by generated print/println calls)
 * Full fmt package is a separate emitted file; these cover builtins.
 * ------------------------------------------------------------------------- */

static inline void hg_print_string(hg_string_t s) {
    if (s.len > 0) fwrite(s.ptr, 1, (size_t)s.len, stdout);
}

static inline void hg_println_string(hg_string_t s) {
    hg_print_string(s);
    fputc('\n', stdout);
}

static inline void hg_print_int64(int64_t v)  { printf("%lld",   (long long)v); }
static inline void hg_print_uint64(uint64_t v) { printf("%llu",  (unsigned long long)v); }
static inline void hg_print_float64(double v)  { printf("%g",    v); }
static inline void hg_print_bool(bool v)        { fputs(v ? "true" : "false", stdout); }

/* ---------------------------------------------------------------------------
 * Runtime initialisation (M0: no-op; later milestones hook GC / scheduler)
 * ------------------------------------------------------------------------- */

static inline void hg_runtime_init(void) {
    /* nothing to do in M0 */
}
