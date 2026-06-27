#pragma once
/* hagane map runtime — open-addressing robin-hood hash table */
#include "hagane_rt.h"

/* ── types ─────────────────────────────────────────────────────────────── */

typedef uint32_t (*hg_hash_fn)(const void *key, uint32_t seed);
typedef bool     (*hg_eq_fn)(const void *a, const void *b);

struct hg_map {
    size_t       key_size;
    size_t       val_size;
    hg_hash_fn   key_hash;
    hg_eq_fn     key_eq;
    int64_t      count;
    int64_t      cap;      /* number of slots; always a power of 2 */
    uint32_t     seed;
    uint8_t     *slots;    /* cap * slot_size bytes; slot = [psl:u8 | key | pad | val] */
    size_t       slot_size;
    size_t       val_off;  /* offset of value within slot */
};

/* hg_map_t is forward-declared as typedef struct hg_map hg_map_t in hagane_rt.h */

typedef struct {
    hg_map_t *m;
    int64_t   pos;
    int64_t   start;
    bool      started;
} hg_map_iter_t;

/* ── API ────────────────────────────────────────────────────────────────── */

hg_map_t* hg_map_new(size_t key_size, size_t val_size,
                     hg_hash_fn key_hash, hg_eq_fn key_eq,
                     int64_t hint);
bool    hg_map_get(hg_map_t *m, const void *key, void *val_out);
void    hg_map_set(hg_map_t *m, const void *key, const void *val);
void    hg_map_delete(hg_map_t *m, const void *key);
int64_t hg_map_len(hg_map_t *m);
void    hg_map_iter_init(hg_map_t *m, hg_map_iter_t *it);
bool    hg_map_iter_next(hg_map_iter_t *it, void *key_out, void *val_out);

/* ── built-in hash functions ────────────────────────────────────────────── */

/* FNV-1a 32-bit over raw bytes */
static inline uint32_t hg_hash_bytes(const void *key, size_t n, uint32_t seed) {
    const uint8_t *p = (const uint8_t*)key;
    uint32_t h = 2166136261u ^ seed;
    for (size_t i = 0; i < n; i++) {
        h ^= p[i];
        h *= 16777619u;
    }
    return h;
}

static inline uint32_t hg_hash_i8 (const void *k, uint32_t s) { return hg_hash_bytes(k, 1, s); }
static inline uint32_t hg_hash_i16(const void *k, uint32_t s) { return hg_hash_bytes(k, 2, s); }
static inline uint32_t hg_hash_i32(const void *k, uint32_t s) { return hg_hash_bytes(k, 4, s); }
static inline uint32_t hg_hash_i64(const void *k, uint32_t s) { return hg_hash_bytes(k, 8, s); }
static inline uint32_t hg_hash_str(const void *k, uint32_t s) {
    const hg_string_t *str = (const hg_string_t*)k;
    return hg_hash_bytes(str->ptr, (size_t)str->len, s);
}
static inline uint32_t hg_hash_ptr(const void *k, uint32_t s) {
    uintptr_t v; memcpy(&v, k, sizeof(v));
    return hg_hash_bytes(&v, sizeof(v), s);
}

static inline bool hg_eq_i8 (const void *a, const void *b) { return *(int8_t*)a  == *(int8_t*)b;  }
static inline bool hg_eq_i16(const void *a, const void *b) { return *(int16_t*)a == *(int16_t*)b; }
static inline bool hg_eq_i32(const void *a, const void *b) { return *(int32_t*)a == *(int32_t*)b; }
static inline bool hg_eq_i64(const void *a, const void *b) { return *(int64_t*)a == *(int64_t*)b; }
static inline bool hg_eq_u8 (const void *a, const void *b) { return *(uint8_t*)a  == *(uint8_t*)b;  }
static inline bool hg_eq_u16(const void *a, const void *b) { return *(uint16_t*)a == *(uint16_t*)b; }
static inline bool hg_eq_u32(const void *a, const void *b) { return *(uint32_t*)a == *(uint32_t*)b; }
static inline bool hg_eq_u64(const void *a, const void *b) { return *(uint64_t*)a == *(uint64_t*)b; }
static inline bool hg_eq_bool(const void *a, const void *b) { return *(bool*)a == *(bool*)b; }
static inline bool hg_eq_str(const void *a, const void *b) {
    const hg_string_t *sa = (const hg_string_t*)a;
    const hg_string_t *sb = (const hg_string_t*)b;
    if (sa->len != sb->len) return false;
    if (sa->len == 0)       return true;
    return memcmp(sa->ptr, sb->ptr, (size_t)sa->len) == 0;
}
static inline bool hg_eq_ptr(const void *a, const void *b) {
    uintptr_t va, vb;
    memcpy(&va, a, sizeof(va));
    memcpy(&vb, b, sizeof(vb));
    return va == vb;
}

/* ── string range iterator ──────────────────────────────────────────────── */
typedef struct { hg_string_t s; int64_t pos; } hg_string_iter_t;
typedef struct { bool r0; int64_t r1; int32_t r2; } hg_string_next_t;

/* decode one UTF-8 rune from iter; advances iter.pos */
static inline hg_string_next_t hg_string_iter_next(hg_string_iter_t *it) {
    if (it->pos >= it->s.len) return (hg_string_next_t){false, 0, 0};
    int64_t p = it->pos;
    unsigned char b = (unsigned char)it->s.ptr[p];
    int32_t r; int64_t sz;
    if      (b < 0x80) { r = b; sz = 1; }
    else if (b < 0xE0) {
        r  = (b & 0x1F) << 6;
        r |= ((unsigned char)it->s.ptr[p+1]) & 0x3F;
        sz = 2;
    } else if (b < 0xF0) {
        r  = (b & 0x0F) << 12;
        r |= (((unsigned char)it->s.ptr[p+1]) & 0x3F) << 6;
        r |= ((unsigned char)it->s.ptr[p+2])  & 0x3F;
        sz = 3;
    } else {
        r  = (b & 0x07) << 18;
        r |= (((unsigned char)it->s.ptr[p+1]) & 0x3F) << 12;
        r |= (((unsigned char)it->s.ptr[p+2]) & 0x3F) << 6;
        r |= ((unsigned char)it->s.ptr[p+3])  & 0x3F;
        sz = 4;
    }
    it->pos += sz;
    return (hg_string_next_t){true, p, r};
}
