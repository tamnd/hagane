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

/* ── fmt stubs ───────────────────────────────────────────────────────────── */

static void hg_iface_print(hg_iface_t v) {
    if (v.itab == HG_TYPE_BOOL)    { printf("%s",  *(bool*)    v.data ? "true" : "false"); return; }
    if (v.itab == HG_TYPE_INT8)    { printf("%d",  (int)*(int8_t*)    v.data); return; }
    if (v.itab == HG_TYPE_INT16)   { printf("%d",  (int)*(int16_t*)   v.data); return; }
    if (v.itab == HG_TYPE_INT32)   { printf("%d",  *(int32_t*)v.data); return; }
    if (v.itab == HG_TYPE_INT64)   { printf("%lld",(long long)*(int64_t*)v.data); return; }
    if (v.itab == HG_TYPE_UINT8)   { printf("%u",  (unsigned)*(uint8_t*)  v.data); return; }
    if (v.itab == HG_TYPE_UINT16)  { printf("%u",  (unsigned)*(uint16_t*) v.data); return; }
    if (v.itab == HG_TYPE_UINT32)  { printf("%u",  *(uint32_t*)v.data); return; }
    if (v.itab == HG_TYPE_UINT64)  { printf("%llu",(unsigned long long)*(uint64_t*)v.data); return; }
    if (v.itab == HG_TYPE_FLOAT32) { printf("%g",  (double)*(float*)   v.data); return; }
    if (v.itab == HG_TYPE_FLOAT64) { printf("%g",  *(double*)v.data); return; }
    if (v.itab == HG_TYPE_STRING)  {
        hg_string_t s = *(hg_string_t*)v.data;
        printf("%.*s", (int)s.len, s.ptr ? s.ptr : "");
        return;
    }
    if (v.itab == HG_TYPE_UINTPTR) { printf("%llu",(unsigned long long)*(uintptr_t*)v.data); return; }
    /* unknown type: print pointer */
    printf("%p", v.data);
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
        p++;
        if (p >= end) break;
        if (*p == '%') { putchar('%'); p++; continue; }
        /* skip flags */
        while (p < end && (*p=='-'||*p=='+'||*p==' '||*p=='#'||*p=='0')) p++;
        /* skip width */
        while (p < end && *p>='0' && *p<='9') p++;
        /* skip precision */
        if (p < end && *p == '.') { p++; while (p < end && *p>='0' && *p<='9') p++; }
        char verb = (p < end) ? *p++ : 'v';
        if (ai >= args.len) { printf("%%!(MISSING)"); continue; }
        hg_iface_t a = args.ptr[ai++];
        switch (verb) {
        case 'd': case 'i':
            if (a.itab==HG_TYPE_INT64||a.itab==HG_TYPE_INT32||a.itab==HG_TYPE_INT16||a.itab==HG_TYPE_INT8)
                printf("%lld", (long long)*(int64_t*)a.data);
            else if (a.itab==HG_TYPE_UINT64||a.itab==HG_TYPE_UINT32||a.itab==HG_TYPE_UINT16||a.itab==HG_TYPE_UINT8)
                printf("%llu", (unsigned long long)*(uint64_t*)a.data);
            else hg_iface_print(a); break;
        case 'u':
            printf("%llu", (unsigned long long)*(uint64_t*)a.data); break;
        case 's':
            if (a.itab==HG_TYPE_STRING) {
                hg_string_t s = *(hg_string_t*)a.data;
                printf("%.*s", (int)s.len, s.ptr ? s.ptr : "");
            } else hg_iface_print(a); break;
        case 'v': hg_iface_print(a); break;
        case 't':
            if (a.itab==HG_TYPE_BOOL) printf("%s", *(bool*)a.data ? "true" : "false");
            else hg_iface_print(a); break;
        case 'f': case 'e': case 'E': case 'g': case 'G':
            if (a.itab==HG_TYPE_FLOAT64) printf("%g", *(double*)a.data);
            else if (a.itab==HG_TYPE_FLOAT32) printf("%g", (double)*(float*)a.data);
            else hg_iface_print(a); break;
        case 'x':
            if (a.itab==HG_TYPE_INT64||a.itab==HG_TYPE_UINT64)
                printf("%llx", (unsigned long long)*(uint64_t*)a.data);
            else hg_iface_print(a); break;
        case 'X':
            if (a.itab==HG_TYPE_INT64||a.itab==HG_TYPE_UINT64)
                printf("%llX", (unsigned long long)*(uint64_t*)a.data);
            else hg_iface_print(a); break;
        case 'o':
            if (a.itab==HG_TYPE_INT64||a.itab==HG_TYPE_UINT64)
                printf("%llo", (unsigned long long)*(uint64_t*)a.data);
            else hg_iface_print(a); break;
        case 'b':
            /* Go %b prints binary; C has no %b — emit manually for ints */
            if (a.itab==HG_TYPE_INT64||a.itab==HG_TYPE_UINT64) {
                uint64_t val = *(uint64_t*)a.data;
                if (val == 0) { putchar('0'); break; }
                char buf[65]; int i = 64;
                buf[i] = 0;
                while (val) { buf[--i] = '0' + (int)(val&1); val >>= 1; }
                printf("%s", buf+i);
            } else hg_iface_print(a); break;
        case 'c':
            if (a.itab==HG_TYPE_INT64) printf("%c", (int)*(int64_t*)a.data);
            else hg_iface_print(a); break;
        case 'p': printf("%p", a.data); break;
        default: putchar('%'); putchar(verb); break;
        }
    }
}
