#include "hagane_rt.h"
#include <stdarg.h>
#include <time.h>
#ifdef _WIN32
#include <io.h>
#include <fcntl.h>
#endif

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

/* ── string conversion helpers ──────────────────────────────────────────── */

hg_slice_uint8_t hg_string_to_bytes(hg_string_t s) {
    if (s.len == 0) return (hg_slice_uint8_t){NULL, 0, 0};
    uint8_t *p = (uint8_t*)hg_alloc((size_t)s.len);
    memcpy(p, s.ptr, (size_t)s.len);
    return (hg_slice_uint8_t){p, s.len, s.len};
}

hg_string_t hg_bytes_to_string(hg_slice_uint8_t b) {
    if (b.len == 0) return HG_ZERO_STRING;
    char *p = (char*)hg_alloc((size_t)b.len);
    memcpy(p, b.ptr, (size_t)b.len);
    return (hg_string_t){p, b.len};
}

/* Decode one UTF-8 sequence from *p (within end). Advances *p and returns rune.
   Returns U+FFFD on invalid bytes. */
static int32_t hg_utf8_decode(const char **p, const char *end) {
    uint8_t b0 = (uint8_t)**p; (*p)++;
    if (b0 < 0x80) return (int32_t)b0;
    if (b0 < 0xC0) return 0xFFFD;
    int32_t r; int n;
    if      (b0 < 0xE0) { r = b0 & 0x1F; n = 1; }
    else if (b0 < 0xF0) { r = b0 & 0x0F; n = 2; }
    else                { r = b0 & 0x07; n = 3; }
    for (int i = 0; i < n; i++) {
        if (*p >= end) return 0xFFFD;
        uint8_t c = (uint8_t)**p;
        if ((c & 0xC0) != 0x80) return 0xFFFD;
        r = (r << 6) | (c & 0x3F); (*p)++;
    }
    return r;
}

hg_slice_int32_t hg_string_to_runes(hg_string_t s) {
    if (s.len == 0) return (hg_slice_int32_t){NULL, 0, 0};
    /* Upper bound: one rune per byte */
    int32_t *buf = (int32_t*)hg_alloc((size_t)s.len * sizeof(int32_t));
    int64_t n = 0;
    const char *p = s.ptr, *end = s.ptr + s.len;
    while (p < end) buf[n++] = hg_utf8_decode(&p, end);
    return (hg_slice_int32_t){buf, n, s.len};
}

/* Encode rune to UTF-8 in buf (must have at least 4 bytes). Returns bytes written. */
static int hg_utf8_encode(int32_t r, char *buf) {
    if (r < 0x80)  { buf[0]=(char)r; return 1; }
    if (r < 0x800) { buf[0]=(char)(0xC0|(r>>6)); buf[1]=(char)(0x80|(r&0x3F)); return 2; }
    if (r < 0x10000){ buf[0]=(char)(0xE0|(r>>12)); buf[1]=(char)(0x80|((r>>6)&0x3F)); buf[2]=(char)(0x80|(r&0x3F)); return 3; }
    buf[0]=(char)(0xF0|(r>>18)); buf[1]=(char)(0x80|((r>>12)&0x3F));
    buf[2]=(char)(0x80|((r>>6)&0x3F)); buf[3]=(char)(0x80|(r&0x3F)); return 4;
}

hg_string_t hg_runes_to_string(hg_slice_int32_t sl) {
    if (sl.len == 0) return HG_ZERO_STRING;
    size_t cap = (size_t)sl.len * 4;
    char *buf = (char*)hg_alloc(cap);
    size_t n = 0;
    for (int64_t i = 0; i < sl.len; i++) n += (size_t)hg_utf8_encode(sl.ptr[i], buf + n);
    /* shrink to actual length */
    if (n < cap) {
        char *s2 = (char*)realloc(buf, n > 0 ? n : 1);
        if (s2) buf = s2;
    }
    return (hg_string_t){buf, (int64_t)n};
}

/* ── misc ────────────────────────────────────────────────────────────────── */

void hg_memmove(void *dst, const void *src, size_t n) {
    memmove(dst, src, n);
}

void hg_runtime_init(void) {
    /* seed for map iteration order randomization */
    srand((unsigned)(time(NULL) ^ (uintptr_t)&hg_runtime_init));
#ifdef _WIN32
    /* Go's runtime sets stdout/stderr to binary mode; match that behavior. */
    _setmode(_fileno(stdout), _O_BINARY);
    _setmode(_fileno(stderr), _O_BINARY);
#endif
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
    case HG_KIND_SLICE: {
        if (!tab->type->elem) { fprintf(f, "[]"); return; }
        const hg_rawslice_t *sl = (const hg_rawslice_t*)v.data;
        const hg_type_t *et = tab->type->elem;
        fprintf(f, "[");
        for (int64_t i = 0; i < sl->len; i++) {
            if (i > 0) fprintf(f, " ");
            const void *ep = (const char*)sl->ptr + (size_t)((uint64_t)i * et->size);
            switch (et->kind) {
            case HG_KIND_BOOL:   fprintf(f, "%s", *(const bool*)ep ? "true" : "false"); break;
            case HG_KIND_INT8:   fprintf(f, "%d", (int)*(const int8_t*)ep); break;
            case HG_KIND_INT16:  fprintf(f, "%d", (int)*(const int16_t*)ep); break;
            case HG_KIND_INT32:  fprintf(f, "%d", *(const int32_t*)ep); break;
            case HG_KIND_INT: case HG_KIND_INT64:
                                 fprintf(f, "%lld", (long long)*(const int64_t*)ep); break;
            case HG_KIND_UINT8:  fprintf(f, "%u", (unsigned)*(const uint8_t*)ep); break;
            case HG_KIND_UINT16: fprintf(f, "%u", (unsigned)*(const uint16_t*)ep); break;
            case HG_KIND_UINT32: fprintf(f, "%u", *(const uint32_t*)ep); break;
            case HG_KIND_UINT: case HG_KIND_UINT64:
                                 fprintf(f, "%llu", (unsigned long long)*(const uint64_t*)ep); break;
            case HG_KIND_FLOAT32: fprintf(f, "%g", (double)*(const float*)ep); break;
            case HG_KIND_FLOAT64: fprintf(f, "%g", *(const double*)ep); break;
            case HG_KIND_STRING: {
                const hg_string_t *s = (const hg_string_t*)ep;
                fprintf(f, "%.*s", (int)s->len, s->ptr ? s->ptr : ""); break;
            }
            default: fprintf(f, "?"); break;
            }
        }
        fprintf(f, "]");
        return;
    }
    default: {
        if (tab->stringer) {
            hg_string_t s = tab->stringer(v.data);
            fprintf(f, "%.*s", (int)s.len, s.ptr ? s.ptr : "");
            return;
        }
        const char *nm = (tab->type && tab->type->name) ? tab->type->name : "?";
        fprintf(f, "<%s Value>", nm);
        return;
    }
    }
}

static void hg_iface_print(hg_iface_t v) { hg_iface_fprint(stdout, v); }

/* ── panic/recover state ─────────────────────────────────────────────────── */

hg_panic_frame_t *hg_panic_top    = NULL;
bool              hg_panic_active  = false;
hg_iface_t        hg_panic_value   = {NULL, NULL};

void hg_throw(hg_iface_t val) {
    hg_panic_active = true;
    hg_panic_value  = val;
    if (hg_panic_top) {
        longjmp(hg_panic_top->buf, 1);
    }
    fprintf(stderr, "goroutine 1 [running]:\npanic: ");
    hg_iface_fprint(stderr, val);
    fprintf(stderr, "\n\ngoroutine 1 [running]:\nmain.main()\n");
    abort();
}

hg_iface_t hg_recover(void) {
    if (!hg_panic_active) return HG_ZERO_IFACE;
    hg_iface_t v   = hg_panic_value;
    hg_panic_active = false;
    hg_panic_value  = HG_ZERO_IFACE;
    return v;
}

void hg_repanic(void) {
    if (!hg_panic_active) return;
    if (hg_panic_top) {
        longjmp(hg_panic_top->buf, 1);
    }
    fprintf(stderr, "goroutine 1 [running]:\npanic: ");
    hg_iface_fprint(stderr, hg_panic_value);
    fprintf(stderr, "\n\ngoroutine 1 [running]:\nmain.main()\n");
    abort();
}

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
        case 'T': {
            const char *tname = tab ? tab->type->name : NULL;
            printf("%s", tname ? tname : "<nil>");
            break;
        }
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

/* ── portable grow-as-you-go string buffer ───────────────────────────────── */

typedef struct { char *data; size_t len; size_t cap; } hg_sbuf_t;

static void hg_sbuf_init(hg_sbuf_t *b) {
    b->data = (char*)malloc(128);
    b->len  = 0;
    b->cap  = b->data ? 128 : 0;
}

static void hg_sbuf_grow(hg_sbuf_t *b, size_t extra) {
    size_t need = b->len + extra + 1;
    if (need <= b->cap) return;
    size_t cap = b->cap ? b->cap * 2 : 128;
    while (cap < need) cap *= 2;
    b->data = (char*)realloc(b->data, cap);
    if (!b->data) { fprintf(stderr, "hagane: out of memory\n"); abort(); }
    b->cap = cap;
}

static void hg_sbuf_writec(hg_sbuf_t *b, char c) {
    hg_sbuf_grow(b, 1);
    b->data[b->len++] = c;
}

static void hg_sbuf_writes(hg_sbuf_t *b, const char *s, size_t n) {
    if (!n) return;
    hg_sbuf_grow(b, n);
    memcpy(b->data + b->len, s, n);
    b->len += n;
}

static void hg_sbuf_printf(hg_sbuf_t *b, const char *fmt, ...) {
    char tmp[128];
    va_list ap;
    va_start(ap, fmt);
    int n = vsnprintf(tmp, sizeof(tmp), fmt, ap);
    va_end(ap);
    if (n <= 0) return;
    if ((size_t)n < sizeof(tmp)) {
        hg_sbuf_writes(b, tmp, (size_t)n);
    } else {
        hg_sbuf_grow(b, (size_t)n);
        va_start(ap, fmt);
        vsnprintf(b->data + b->len, (size_t)n + 1, fmt, ap);
        va_end(ap);
        b->len += (size_t)n;
    }
}

static void hg_iface_sbuf(hg_sbuf_t *b, hg_iface_t v) {
    if (!v.itab) { hg_sbuf_writes(b, "<nil>", 5); return; }
    const hg_iface_tab_t *tab = (const hg_iface_tab_t*)v.itab;
    switch (tab->type->kind) {
    case HG_KIND_BOOL:   hg_sbuf_writes(b, *(bool*)v.data ? "true" : "false",
                                        *(bool*)v.data ? 4 : 5); return;
    case HG_KIND_INT8:   hg_sbuf_printf(b, "%d",   (int)*(int8_t*)v.data);   return;
    case HG_KIND_INT16:  hg_sbuf_printf(b, "%d",   (int)*(int16_t*)v.data);  return;
    case HG_KIND_INT32:  hg_sbuf_printf(b, "%d",   *(int32_t*)v.data);       return;
    case HG_KIND_INT: case HG_KIND_INT64:
                         hg_sbuf_printf(b, "%lld",  (long long)*(int64_t*)v.data);  return;
    case HG_KIND_UINT8:  hg_sbuf_printf(b, "%u",   (unsigned)*(uint8_t*)v.data);    return;
    case HG_KIND_UINT16: hg_sbuf_printf(b, "%u",   (unsigned)*(uint16_t*)v.data);   return;
    case HG_KIND_UINT32: hg_sbuf_printf(b, "%u",   *(uint32_t*)v.data);             return;
    case HG_KIND_UINT: case HG_KIND_UINT64:
                         hg_sbuf_printf(b, "%llu",  (unsigned long long)*(uint64_t*)v.data); return;
    case HG_KIND_FLOAT32: hg_sbuf_printf(b, "%g",  (double)*(float*)v.data);        return;
    case HG_KIND_FLOAT64: hg_sbuf_printf(b, "%g",  *(double*)v.data);               return;
    case HG_KIND_STRING: {
        hg_string_t s = *(hg_string_t*)v.data;
        hg_sbuf_writes(b, s.ptr ? s.ptr : "", s.ptr ? (size_t)s.len : 0);
        return;
    }
    case HG_KIND_UINTPTR: hg_sbuf_printf(b, "%llu", (unsigned long long)*(uintptr_t*)v.data); return;
    case HG_KIND_SLICE: {
        if (!tab->type->elem) { hg_sbuf_writes(b, "[]", 2); return; }
        const hg_rawslice_t *sl = (const hg_rawslice_t*)v.data;
        const hg_type_t *et = tab->type->elem;
        hg_sbuf_writec(b, '[');
        for (int64_t i = 0; i < sl->len; i++) {
            if (i > 0) hg_sbuf_writec(b, ' ');
            const void *ep = (const char*)sl->ptr + (size_t)((uint64_t)i * et->size);
            switch (et->kind) {
            case HG_KIND_BOOL:   hg_sbuf_writes(b, *(const bool*)ep ? "true":"false",
                                               *(const bool*)ep ? 4:5); break;
            case HG_KIND_INT8:   hg_sbuf_printf(b, "%d", (int)*(const int8_t*)ep); break;
            case HG_KIND_INT16:  hg_sbuf_printf(b, "%d", (int)*(const int16_t*)ep); break;
            case HG_KIND_INT32:  hg_sbuf_printf(b, "%d", *(const int32_t*)ep); break;
            case HG_KIND_INT: case HG_KIND_INT64:
                                 hg_sbuf_printf(b, "%lld", (long long)*(const int64_t*)ep); break;
            case HG_KIND_UINT8:  hg_sbuf_printf(b, "%u", (unsigned)*(const uint8_t*)ep); break;
            case HG_KIND_UINT16: hg_sbuf_printf(b, "%u", (unsigned)*(const uint16_t*)ep); break;
            case HG_KIND_UINT32: hg_sbuf_printf(b, "%u", *(const uint32_t*)ep); break;
            case HG_KIND_UINT: case HG_KIND_UINT64:
                                 hg_sbuf_printf(b, "%llu", (unsigned long long)*(const uint64_t*)ep); break;
            case HG_KIND_FLOAT32: hg_sbuf_printf(b, "%g", (double)*(const float*)ep); break;
            case HG_KIND_FLOAT64: hg_sbuf_printf(b, "%g", *(const double*)ep); break;
            case HG_KIND_STRING: {
                const hg_string_t *s = (const hg_string_t*)ep;
                hg_sbuf_writes(b, s->ptr ? s->ptr : "", s->ptr ? (size_t)s->len : 0); break;
            }
            default: hg_sbuf_writec(b, '?'); break;
            }
        }
        hg_sbuf_writec(b, ']');
        return;
    }
    default: {
        if (tab->stringer) {
            hg_string_t s = tab->stringer(v.data);
            hg_sbuf_writes(b, s.ptr ? s.ptr : "", s.ptr ? (size_t)s.len : 0);
            return;
        }
        const char *name = (tab->type && tab->type->name) ? tab->type->name : "?";
        hg_sbuf_writes(b, "<", 1);
        hg_sbuf_writes(b, name, strlen(name));
        hg_sbuf_writes(b, " Value>", 7);
        return;
    }
    }
}

/* hg_fmt_sprintf: like hg_fmt_printf but returns the result as hg_string_t */
hg_string_t hg_fmt_sprintf(hg_string_t fmt_str, hg_slice_hg_iface_t_t args) {
    hg_sbuf_t b;
    hg_sbuf_init(&b);

    const char *p   = fmt_str.ptr;
    const char *end = p + fmt_str.len;
    int64_t ai = 0;
    while (p < end) {
        if (*p != '%') { hg_sbuf_writec(&b, *p++); continue; }
        const char *spec_start = p;
        p++;
        if (p >= end) break;
        if (*p == '%') { hg_sbuf_writec(&b, '%'); p++; continue; }
        while (p < end && (*p=='-'||*p=='+'||*p==' '||*p=='#'||*p=='0')) p++;
        while (p < end && *p>='0' && *p<='9') p++;
        if (p < end && *p == '.') { p++; while (p < end && *p>='0' && *p<='9') p++; }
        char verb = (p < end) ? *p++ : 'v';
        char cfmt[64];
        int spec_len = (int)(p - spec_start);
        if (spec_len > 62) spec_len = 62;
        memcpy(cfmt, spec_start, spec_len);
        cfmt[spec_len] = '\0';
        if (ai >= args.len) { hg_sbuf_writes(&b, "%(MISSING)", 10); continue; }
        hg_iface_t a = args.ptr[ai++];
        const hg_iface_tab_t *tab = a.itab ? (const hg_iface_tab_t*)a.itab : NULL;
        uint8_t kind = tab ? tab->type->kind : 0;
        switch (verb) {
        case 'd': case 'i':
            switch (kind) {
            case HG_KIND_INT8:  hg_sbuf_printf(&b, "%lld", (long long)*(int8_t*)a.data);  break;
            case HG_KIND_INT16: hg_sbuf_printf(&b, "%lld", (long long)*(int16_t*)a.data); break;
            case HG_KIND_INT32: hg_sbuf_printf(&b, "%lld", (long long)*(int32_t*)a.data); break;
            case HG_KIND_INT: case HG_KIND_INT64:
                                hg_sbuf_printf(&b, "%lld", (long long)*(int64_t*)a.data); break;
            case HG_KIND_UINT8:  hg_sbuf_printf(&b, "%llu", (unsigned long long)*(uint8_t*)a.data);  break;
            case HG_KIND_UINT16: hg_sbuf_printf(&b, "%llu", (unsigned long long)*(uint16_t*)a.data); break;
            case HG_KIND_UINT32: hg_sbuf_printf(&b, "%llu", (unsigned long long)*(uint32_t*)a.data); break;
            case HG_KIND_UINT: case HG_KIND_UINT64:
                                 hg_sbuf_printf(&b, "%llu", (unsigned long long)*(uint64_t*)a.data); break;
            default: hg_iface_sbuf(&b, a); break;
            } break;
        case 's':
            if (kind == HG_KIND_STRING) {
                hg_string_t s = *(hg_string_t*)a.data;
                hg_sbuf_writes(&b, s.ptr ? s.ptr : "", s.ptr ? (size_t)s.len : 0);
            } else hg_iface_sbuf(&b, a);
            break;
        case 'v': hg_iface_sbuf(&b, a); break;
        case 'T': {
            const char *tname = tab ? tab->type->name : NULL;
            if (tname) hg_sbuf_writes(&b, tname, strlen(tname));
            else hg_sbuf_writes(&b, "<nil>", 5);
            break;
        }
        case 'f': case 'e': case 'E': case 'g': case 'G':
            if (kind == HG_KIND_FLOAT64) hg_sbuf_printf(&b, cfmt, *(double*)a.data);
            else if (kind == HG_KIND_FLOAT32) hg_sbuf_printf(&b, cfmt, (double)*(float*)a.data);
            else hg_iface_sbuf(&b, a);
            break;
        default: hg_sbuf_writec(&b, '%'); hg_sbuf_writec(&b, verb); break;
        }
    }

    /* NUL-terminate and return as hg_string_t (transfers ownership of b.data) */
    hg_sbuf_grow(&b, 1);
    b.data[b.len] = '\0';
    return (hg_string_t){.ptr = b.data, .len = (int64_t)b.len};
}

/* ── errors shim ─────────────────────────────────────────────────────────── */

typedef struct { hg_string_t msg; } hg_errors_errorString_t;

static hg_string_t hg_errors_errorString_Error(void *self) {
    return ((hg_errors_errorString_t*)self)->msg;
}

static const hg_type_t hg_type_errors_errorString = {
    sizeof(hg_errors_errorString_t), HG_KIND_STRUCT, "*errors.errorString", NULL
};
static void *hg_errors_errorString_methods[] = { (void*)hg_errors_errorString_Error };
static const hg_iface_tab_t hg_itab_errors_errorString = {
    &hg_type_errors_errorString,
    hg_errors_errorString_methods,
    hg_errors_errorString_Error,
};

hg_iface_t hg_errors_New(hg_string_t msg) {
    hg_errors_errorString_t *s = (hg_errors_errorString_t*)hg_alloc(sizeof(hg_errors_errorString_t));
    s->msg = msg;
    return (hg_iface_t){.itab = (const void*)&hg_itab_errors_errorString, .data = s};
}
