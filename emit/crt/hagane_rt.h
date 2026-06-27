#pragma once
/* hagane runtime — included by all emitted C files */
#include <stdint.h>
#include <stdbool.h>
#include <stddef.h>
#include <stdlib.h>
#include <string.h>
#include <stdio.h>

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

/* ── M0 type tags (encoded in itab for interface boxing) ─────────────────── */
#define HG_TYPE_UNKNOWN  ((const void*)0)
#define HG_TYPE_BOOL     ((const void*)1)
#define HG_TYPE_INT8     ((const void*)2)
#define HG_TYPE_INT16    ((const void*)3)
#define HG_TYPE_INT32    ((const void*)4)
#define HG_TYPE_INT64    ((const void*)5)
#define HG_TYPE_UINT8    ((const void*)6)
#define HG_TYPE_UINT16   ((const void*)7)
#define HG_TYPE_UINT32   ((const void*)8)
#define HG_TYPE_UINT64   ((const void*)9)
#define HG_TYPE_FLOAT32  ((const void*)10)
#define HG_TYPE_FLOAT64  ((const void*)11)
#define HG_TYPE_STRING   ((const void*)12)
#define HG_TYPE_UINTPTR  ((const void*)13)

/* ── runtime functions ───────────────────────────────────────────────────── */
void*        hg_alloc(size_t size);
void*        hg_realloc(void *ptr, size_t old_size, size_t new_size);
hg_string_t  hg_string_concat(hg_string_t a, hg_string_t b);
bool         hg_string_equal(hg_string_t a, hg_string_t b);
int          hg_string_compare(hg_string_t a, hg_string_t b);
void*        hg_makeslice_raw(size_t elem_size, int64_t len, int64_t cap);
void*        hg_growslice_raw(size_t elem_size, void *old_ptr, int64_t len, int64_t *cap_out, int64_t extra);
void         hg_memmove(void *dst, const void *src, size_t n);
void         hg_runtime_init(void);

/* fmt stubs (called from generated init functions) */
static inline void hg_fmt_init(void) {}

/* fmt.Print* via type-tagged interfaces */
void hg_fmt_println(hg_slice_hg_iface_t_t args);
void hg_fmt_print(hg_slice_hg_iface_t_t args);
void hg_fmt_printf(hg_string_t fmt, hg_slice_hg_iface_t_t args);

/* helper: print a Go string via printf */
static inline void hg_print_string(hg_string_t s) {
    if (s.ptr && s.len > 0) {
        fwrite(s.ptr, 1, (size_t)s.len, stdout);
    }
}
