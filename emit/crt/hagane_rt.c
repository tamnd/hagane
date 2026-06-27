#include "hagane_rt.h"
#include <time.h>

/* ── memory ──────────────────────────────────────────────────────────────── */

void* hg_alloc(size_t size) {
    void *p = calloc(1, size > 0 ? size : 1);
    if (!p) { fprintf(stderr, "hagane: out of memory\n"); abort(); }
    return p;
}

void* hg_realloc(void *ptr, size_t old_size, size_t new_size) {
    void *p = realloc(ptr, new_size > 0 ? new_size : 1);
    if (!p) { fprintf(stderr, "hagane: out of memory\n"); abort(); }
    if (new_size > old_size)
        memset((char*)p + old_size, 0, new_size - old_size);
    return p;
}

/* ── string ──────────────────────────────────────────────────────────────── */

hg_string_t hg_string_concat(hg_string_t a, hg_string_t b) {
    int64_t total = a.len + b.len;
    if (total == 0) return HG_ZERO_STRING;
    char *buf = (char*)hg_alloc((size_t)total);
    if (a.len > 0) memcpy(buf,          a.ptr, (size_t)a.len);
    if (b.len > 0) memcpy(buf + a.len,  b.ptr, (size_t)b.len);
    return (hg_string_t){.ptr = buf, .len = total};
}

bool hg_string_equal(hg_string_t a, hg_string_t b) {
    if (a.len != b.len) return false;
    if (a.len == 0)     return true;
    return memcmp(a.ptr, b.ptr, (size_t)a.len) == 0;
}

int hg_string_compare(hg_string_t a, hg_string_t b) {
    size_t n = (size_t)(a.len < b.len ? a.len : b.len);
    int cmp = memcmp(a.ptr, b.ptr, n);
    if (cmp != 0) return cmp;
    if (a.len < b.len) return -1;
    if (a.len > b.len) return  1;
    return 0;
}

/* ── slice ───────────────────────────────────────────────────────────────── */

void* hg_makeslice_raw(size_t elem_size, int64_t len, int64_t cap) {
    if (len < 0 || cap < len) {
        fprintf(stderr, "hagane: runtime error: makeslice: len out of range\n");
        abort();
    }
    if (cap == 0) return NULL;
    /* check for multiplication overflow */
    if (elem_size > 0 && (size_t)cap > (size_t)(-1) / elem_size) {
        fprintf(stderr, "hagane: runtime error: makeslice: cap out of range\n");
        abort();
    }
    return hg_alloc((size_t)cap * elem_size);
}

void* hg_growslice_raw(size_t elem_size, void *old_ptr, int64_t len, int64_t *cap_out, int64_t extra) {
    int64_t new_cap = *cap_out * 2;
    if (new_cap < len + extra) new_cap = len + extra;
    if (new_cap < 4)           new_cap = 4;
    size_t old_bytes = (size_t)(*cap_out) * elem_size;
    size_t new_bytes = (size_t)new_cap   * elem_size;
    void *p = realloc(old_ptr, new_bytes);
    if (!p) { fprintf(stderr, "hagane: out of memory\n"); abort(); }
    if (new_bytes > old_bytes)
        memset((char*)p + old_bytes, 0, new_bytes - old_bytes);
    *cap_out = new_cap;
    return p;
}

/* ── slice helpers (internal; generated code emits inline instead) ───────── */

static hg_slice_uint8_t hg_append_one(hg_slice_uint8_t s, const void *elem, size_t elem_size) {
    if (s.len >= s.cap) {
        int64_t new_cap = s.cap * 2;
        if (new_cap < s.len + 1) new_cap = s.len + 1;
        if (new_cap < 4)         new_cap = 4;
        size_t old_bytes = (size_t)s.cap * elem_size;
        size_t new_bytes = (size_t)new_cap * elem_size;
        uint8_t *p = (uint8_t*)realloc(s.ptr, new_bytes);
        if (!p) { fprintf(stderr, "hagane: out of memory\n"); abort(); }
        if (new_bytes > old_bytes) memset(p + old_bytes, 0, new_bytes - old_bytes);
        s.ptr = p; s.cap = new_cap;
    }
    memcpy(s.ptr + (size_t)s.len * elem_size, elem, elem_size);
    s.len++;
    return s;
}

static hg_slice_uint8_t hg_append_slice(hg_slice_uint8_t dst, hg_slice_uint8_t src, size_t elem_size) {
    if (src.len == 0) return dst;
    int64_t need = dst.len + src.len;
    if (need > dst.cap) {
        int64_t new_cap = dst.cap * 2;
        if (new_cap < need) new_cap = need;
        size_t old_bytes = (size_t)dst.cap * elem_size;
        size_t new_bytes = (size_t)new_cap * elem_size;
        uint8_t *p = (uint8_t*)realloc(dst.ptr, new_bytes);
        if (!p) { fprintf(stderr, "hagane: out of memory\n"); abort(); }
        if (new_bytes > old_bytes) memset(p + old_bytes, 0, new_bytes - old_bytes);
        dst.ptr = p; dst.cap = new_cap;
    }
    memcpy(dst.ptr + (size_t)dst.len * elem_size, src.ptr, (size_t)src.len * elem_size);
    dst.len += src.len;
    return dst;
}

static int64_t hg_copy_slice(hg_slice_uint8_t dst, hg_slice_uint8_t src, size_t elem_size) {
    int64_t n = dst.len < src.len ? dst.len : src.len;
    if (n > 0) memmove(dst.ptr, src.ptr, (size_t)n * elem_size);
    return n;
}

/* ── misc ────────────────────────────────────────────────────────────────── */

void hg_memmove(void *dst, const void *src, size_t n) {
    memmove(dst, src, n);
}

void hg_runtime_init(void) {
    /* seed for map iteration order randomization */
    srand((unsigned)(time(NULL) ^ (uintptr_t)&hg_runtime_init));
}

void hg_panic_typeassert(const char *have, const char *want) {
    fprintf(stderr, "interface conversion: interface {} is %s, not %s\n", have, want);
    abort();
}

void hg_panic_iface_nil(const char *pos) {
    (void)pos;
    fprintf(stderr, "goroutine 1 [running]:\nruntime error: invalid memory address or nil pointer dereference\n");
    abort();
}

/* ── M3 type descriptors for primitive types ─────────────────────────────── */

const hg_type_t hg_type_bool    = {sizeof(bool),    HG_KIND_BOOL,    "bool"};
const hg_type_t hg_type_int8    = {sizeof(int8_t),  HG_KIND_INT8,    "int8"};
const hg_type_t hg_type_int16   = {sizeof(int16_t), HG_KIND_INT16,   "int16"};
const hg_type_t hg_type_int32   = {sizeof(int32_t), HG_KIND_INT32,   "int32"};
const hg_type_t hg_type_int64   = {sizeof(int64_t), HG_KIND_INT64,   "int"};
const hg_type_t hg_type_uint8   = {sizeof(uint8_t), HG_KIND_UINT8,   "uint8"};
const hg_type_t hg_type_uint16  = {sizeof(uint16_t),HG_KIND_UINT16,  "uint16"};
const hg_type_t hg_type_uint32  = {sizeof(uint32_t),HG_KIND_UINT32,  "uint32"};
const hg_type_t hg_type_uint64  = {sizeof(uint64_t),HG_KIND_UINT64,  "uint"};
const hg_type_t hg_type_float32 = {sizeof(float),   HG_KIND_FLOAT32, "float32"};
const hg_type_t hg_type_float64 = {sizeof(double),  HG_KIND_FLOAT64, "float64"};
const hg_type_t hg_type_string  = {sizeof(hg_string_t), HG_KIND_STRING, "string"};
const hg_type_t hg_type_uintptr = {sizeof(uintptr_t), HG_KIND_UINTPTR, "uintptr"};

/* Primitive itab singletons: concrete type with no interface methods */
const hg_iface_tab_t hg_itab_bool    = {&hg_type_bool,    NULL};
const hg_iface_tab_t hg_itab_int8    = {&hg_type_int8,    NULL};
const hg_iface_tab_t hg_itab_int16   = {&hg_type_int16,   NULL};
const hg_iface_tab_t hg_itab_int32   = {&hg_type_int32,   NULL};
const hg_iface_tab_t hg_itab_int64   = {&hg_type_int64,   NULL};
const hg_iface_tab_t hg_itab_uint8   = {&hg_type_uint8,   NULL};
const hg_iface_tab_t hg_itab_uint16  = {&hg_type_uint16,  NULL};
const hg_iface_tab_t hg_itab_uint32  = {&hg_type_uint32,  NULL};
const hg_iface_tab_t hg_itab_uint64  = {&hg_type_uint64,  NULL};
const hg_iface_tab_t hg_itab_float32 = {&hg_type_float32, NULL};
const hg_iface_tab_t hg_itab_float64 = {&hg_type_float64, NULL};
const hg_iface_tab_t hg_itab_string  = {&hg_type_string,  NULL};
const hg_iface_tab_t hg_itab_uintptr = {&hg_type_uintptr, NULL};

/* ── fmt stubs ───────────────────────────────────────────────────────────── */

static void hg_iface_fprint(FILE *f, hg_iface_t v) {
    if (!v.itab) { fprintf(f, "<nil>"); return; }
    const hg_iface_tab_t *tab = (const hg_iface_tab_t*)v.itab;
    switch (tab->type->kind) {
    case HG_KIND_BOOL:    fprintf(f, "%s", *(bool*)v.data ? "true" : "false"); return;
    case HG_KIND_INT8:    fprintf(f, "%d", (int)*(int8_t*)v.data); return;
    case HG_KIND_INT16:   fprintf(f, "%d", (int)*(int16_t*)v.data); return;
    case HG_KIND_INT32:   fprintf(f, "%d", *(int32_t*)v.data); return;
    case HG_KIND_INT:
    case HG_KIND_INT64:   fprintf(f, "%lld", (long long)*(int64_t*)v.data); return;
    case HG_KIND_UINT8:   fprintf(f, "%u", (unsigned)*(uint8_t*)v.data); return;
    case HG_KIND_UINT16:  fprintf(f, "%u", (unsigned)*(uint16_t*)v.data); return;
    case HG_KIND_UINT32:  fprintf(f, "%u", *(uint32_t*)v.data); return;
    case HG_KIND_UINT:
    case HG_KIND_UINT64:  fprintf(f, "%llu", (unsigned long long)*(uint64_t*)v.data); return;
    case HG_KIND_FLOAT32: fprintf(f, "%g", (double)*(float*)v.data); return;
    case HG_KIND_FLOAT64: fprintf(f, "%g", *(double*)v.data); return;
    case HG_KIND_STRING: {
        hg_string_t s = *(hg_string_t*)v.data;
        fprintf(f, "%.*s", (int)s.len, s.ptr ? s.ptr : "");
        return;
    }
    case HG_KIND_UINTPTR: fprintf(f, "%llu", (unsigned long long)*(uintptr_t*)v.data); return;
    default: fprintf(f, "<%s Value>", tab->type->name ? tab->type->name : "?"); return;
    }
}

static void hg_iface_print(hg_iface_t v) { hg_iface_fprint(stdout, v); }

void hg_fmt_println(hg_slice_hg_iface_t_t args) {
    for (int64_t i = 0; i < args.len; i++) {
        if (i > 0) putchar(' ');
        hg_iface_print(args.ptr[i]);
    }
    putchar('\n');
}

void hg_fmt_print(hg_slice_hg_iface_t_t args) {
    for (int64_t i = 0; i < args.len; i++) {
        hg_iface_print(args.ptr[i]);
    }
}

void hg_fmt_printf(hg_string_t fmt_str, hg_slice_hg_iface_t_t args) {
    const char *p   = fmt_str.ptr;
    const char *end = p + fmt_str.len;
    int64_t ai = 0;
    while (p < end) {
        if (*p != '%') { putchar(*p++); continue; }
        const char *spec_start = p; /* points at '%' */
        p++;
        if (p >= end) break;
        if (*p == '%') { putchar('%'); p++; continue; }
        /* collect flags, width, precision */
        while (p < end && (*p=='-'||*p=='+'||*p==' '||*p=='#'||*p=='0')) p++;
        while (p < end && *p>='0' && *p<='9') p++;
        if (p < end && *p == '.') { p++; while (p < end && *p>='0' && *p<='9') p++; }
        char verb = (p < end) ? *p++ : 'v';
        /* build NUL-terminated C format spec from spec_start..p */
        char cfmt[64];
        int spec_len = (int)(p - spec_start);
        if (spec_len > 62) spec_len = 62;
        memcpy(cfmt, spec_start, spec_len);
        cfmt[spec_len] = '\0';
        if (ai >= args.len) { printf("%%!(MISSING)"); continue; }
        hg_iface_t a = args.ptr[ai++];
        const hg_iface_tab_t *tab = a.itab ? (const hg_iface_tab_t*)a.itab : NULL;
        uint8_t kind = tab ? tab->type->kind : 0;
        switch (verb) {
        case 'd': case 'i':
            switch (kind) {
            case HG_KIND_INT8:  printf("%lld", (long long)*(int8_t*)a.data);  break;
            case HG_KIND_INT16: printf("%lld", (long long)*(int16_t*)a.data); break;
            case HG_KIND_INT32: printf("%lld", (long long)*(int32_t*)a.data); break;
            case HG_KIND_INT: case HG_KIND_INT64:
                                printf("%lld", (long long)*(int64_t*)a.data); break;
            case HG_KIND_UINT8:  printf("%llu", (unsigned long long)*(uint8_t*)a.data);  break;
            case HG_KIND_UINT16: printf("%llu", (unsigned long long)*(uint16_t*)a.data); break;
            case HG_KIND_UINT32: printf("%llu", (unsigned long long)*(uint32_t*)a.data); break;
            case HG_KIND_UINT: case HG_KIND_UINT64:
                                 printf("%llu", (unsigned long long)*(uint64_t*)a.data); break;
            default: hg_iface_print(a); break;
            } break;
        case 'u':
            printf("%llu", (unsigned long long)*(uint64_t*)a.data); break;
        case 's':
            if (kind == HG_KIND_STRING) {
                hg_string_t s = *(hg_string_t*)a.data;
                printf("%.*s", (int)s.len, s.ptr ? s.ptr : "");
            } else hg_iface_print(a); break;
        case 'v': hg_iface_print(a); break;
        case 't':
            if (kind == HG_KIND_BOOL) printf("%s", *(bool*)a.data ? "true" : "false");
            else hg_iface_print(a); break;
        case 'f': case 'e': case 'E': case 'g': case 'G':
            if (kind == HG_KIND_FLOAT64) printf(cfmt, *(double*)a.data);
            else if (kind == HG_KIND_FLOAT32) printf(cfmt, (double)*(float*)a.data);
            else hg_iface_print(a); break;
        case 'x':
            switch (kind) {
            case HG_KIND_INT: case HG_KIND_INT64: case HG_KIND_UINT: case HG_KIND_UINT64:
                printf("%llx", (unsigned long long)*(uint64_t*)a.data); break;
            default: hg_iface_print(a); break;
            } break;
        case 'X':
            switch (kind) {
            case HG_KIND_INT: case HG_KIND_INT64: case HG_KIND_UINT: case HG_KIND_UINT64:
                printf("%llX", (unsigned long long)*(uint64_t*)a.data); break;
            default: hg_iface_print(a); break;
            } break;
        case 'o':
            switch (kind) {
            case HG_KIND_INT: case HG_KIND_INT64: case HG_KIND_UINT: case HG_KIND_UINT64:
                printf("%llo", (unsigned long long)*(uint64_t*)a.data); break;
            default: hg_iface_print(a); break;
            } break;
        case 'b':
            if (kind==HG_KIND_INT||kind==HG_KIND_INT64||kind==HG_KIND_UINT||kind==HG_KIND_UINT64) {
                uint64_t val = *(uint64_t*)a.data;
                if (val == 0) { putchar('0'); break; }
                char buf[65]; int bi = 64;
                buf[bi] = 0;
                while (val) { buf[--bi] = '0' + (int)(val&1); val >>= 1; }
                printf("%s", buf+bi);
            } else hg_iface_print(a); break;
        case 'c':
            if (kind==HG_KIND_INT||kind==HG_KIND_INT64) printf("%c", (int)*(int64_t*)a.data);
            else hg_iface_print(a); break;
        case 'p': printf("%p", a.data); break;
        default: putchar('%'); putchar(verb); break;
        }
    }
}

/* hg_fmt_sprintf: like hg_fmt_printf but returns the result as hg_string_t */
hg_string_t hg_fmt_sprintf(hg_string_t fmt_str, hg_slice_hg_iface_t_t args) {
    char *buf = NULL;
    size_t buf_len = 0;
    FILE *out = open_memstream(&buf, &buf_len);
    if (!out) return HG_ZERO_STRING;

    const char *p   = fmt_str.ptr;
    const char *end = p + fmt_str.len;
    int64_t ai = 0;
    while (p < end) {
        if (*p != '%') { fputc(*p++, out); continue; }
        const char *spec_start = p;
        p++;
        if (p >= end) break;
        if (*p == '%') { fputc('%', out); p++; continue; }
        while (p < end && (*p=='-'||*p=='+'||*p==' '||*p=='#'||*p=='0')) p++;
        while (p < end && *p>='0' && *p<='9') p++;
        if (p < end && *p == '.') { p++; while (p < end && *p>='0' && *p<='9') p++; }
        char verb = (p < end) ? *p++ : 'v';
        char cfmt[64];
        int spec_len = (int)(p - spec_start);
        if (spec_len > 62) spec_len = 62;
        memcpy(cfmt, spec_start, spec_len);
        cfmt[spec_len] = '\0';
        if (ai >= args.len) { fprintf(out, "%%!(MISSING)"); continue; }
        hg_iface_t a = args.ptr[ai++];
        const hg_iface_tab_t *tab = a.itab ? (const hg_iface_tab_t*)a.itab : NULL;
        uint8_t kind = tab ? tab->type->kind : 0;
        switch (verb) {
        case 'd': case 'i':
            switch (kind) {
            case HG_KIND_INT8:  fprintf(out, "%lld", (long long)*(int8_t*)a.data);  break;
            case HG_KIND_INT16: fprintf(out, "%lld", (long long)*(int16_t*)a.data); break;
            case HG_KIND_INT32: fprintf(out, "%lld", (long long)*(int32_t*)a.data); break;
            case HG_KIND_INT: case HG_KIND_INT64:
                                fprintf(out, "%lld", (long long)*(int64_t*)a.data); break;
            case HG_KIND_UINT8:  fprintf(out, "%llu", (unsigned long long)*(uint8_t*)a.data);  break;
            case HG_KIND_UINT16: fprintf(out, "%llu", (unsigned long long)*(uint16_t*)a.data); break;
            case HG_KIND_UINT32: fprintf(out, "%llu", (unsigned long long)*(uint32_t*)a.data); break;
            case HG_KIND_UINT: case HG_KIND_UINT64:
                                 fprintf(out, "%llu", (unsigned long long)*(uint64_t*)a.data); break;
            default: hg_iface_fprint(out, a); break;
            } break;
        case 's':
            if (kind == HG_KIND_STRING) {
                hg_string_t s = *(hg_string_t*)a.data;
                fprintf(out, "%.*s", (int)s.len, s.ptr ? s.ptr : "");
            } else hg_iface_fprint(out, a);
            break;
        case 'v': hg_iface_fprint(out, a); break;
        case 'f': case 'e': case 'E': case 'g': case 'G':
            if (kind == HG_KIND_FLOAT64) fprintf(out, cfmt, *(double*)a.data);
            else if (kind == HG_KIND_FLOAT32) fprintf(out, cfmt, (double)*(float*)a.data);
            else hg_iface_fprint(out, a);
            break;
        default: fputc('%', out); fputc(verb, out); break;
        }
    }
    fclose(out);

    char *result = (char*)hg_alloc(buf_len + 1);
    memcpy(result, buf, buf_len);
    result[buf_len] = '\0';
    free(buf);
    return (hg_string_t){.ptr = result, .len = (int64_t)buf_len};
}
