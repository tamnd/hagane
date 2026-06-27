#include "hagane_map.h"
#include <time.h>

/* ── internal slot access ───────────────────────────────────────────────── */

#define SLOT_EMPTY     0x00
#define SLOT_TOMBSTONE 0xFF

static inline uint8_t *slot_at(hg_map_t *m, int64_t i) {
    return m->slots + (size_t)i * m->slot_size;
}
static inline uint8_t  slot_psl(hg_map_t *m, int64_t i) { return slot_at(m, i)[0]; }
static inline void    *slot_key(hg_map_t *m, int64_t i) { return slot_at(m, i) + 1; }
static inline void    *slot_val(hg_map_t *m, int64_t i) { return slot_at(m, i) + m->val_off; }

/* ── grow / rehash ──────────────────────────────────────────────────────── */

static void hg_map_rehash(hg_map_t *m, int64_t new_cap) {
    uint8_t *old_slots = m->slots;
    int64_t  old_cap   = m->cap;

    m->cap   = new_cap;
    m->count = 0;
    m->slots = (uint8_t*)calloc((size_t)new_cap, m->slot_size);
    if (!m->slots) { fprintf(stderr, "hagane: out of memory\n"); abort(); }

    for (int64_t i = 0; i < old_cap; i++) {
        uint8_t *s = old_slots + (size_t)i * m->slot_size;
        if (s[0] == SLOT_EMPTY || s[0] == SLOT_TOMBSTONE) continue;
        hg_map_set(m, s + 1, s + m->val_off);
    }
    free(old_slots);
}

/* ── public API ─────────────────────────────────────────────────────────── */

hg_map_t* hg_map_new(size_t key_size, size_t val_size,
                     hg_hash_fn key_hash, hg_eq_fn key_eq,
                     int64_t hint) {
    hg_map_t *m = (hg_map_t*)calloc(1, sizeof(hg_map_t));
    if (!m) { fprintf(stderr, "hagane: out of memory\n"); abort(); }

    /* align val_off to 8 bytes to avoid unaligned access */
    size_t val_off = ((1 + key_size) + 7) & ~(size_t)7;
    size_t slot_size = val_off + val_size;
    /* round slot_size to 8 */
    slot_size = (slot_size + 7) & ~(size_t)7;

    m->key_size  = key_size;
    m->val_size  = val_size;
    m->key_hash  = key_hash;
    m->key_eq    = key_eq;
    m->val_off   = val_off;
    m->slot_size = slot_size;

    int64_t cap = 8;
    if (hint > 0) {
        /* pick smallest power of 2 such that load < 0.75 */
        int64_t needed = (hint * 4) / 3 + 1;
        while (cap < needed) cap <<= 1;
    }
    m->cap   = cap;
    m->slots = (uint8_t*)calloc((size_t)cap, slot_size);
    if (!m->slots) { fprintf(stderr, "hagane: out of memory\n"); abort(); }

    /* random seed from time for iteration order randomization */
    m->seed = (uint32_t)((uintptr_t)m ^ (uintptr_t)time(NULL));
    return m;
}

bool hg_map_get(hg_map_t *m, const void *key, void *val_out) {
    if (m->count == 0) return false;
    uint32_t h = m->key_hash(key, m->seed);
    int64_t  mask = m->cap - 1;
    int64_t  i = (int64_t)(h & (uint32_t)mask);
    uint8_t  psl = 1;

    for (;;) {
        uint8_t sp = slot_psl(m, i);
        if (sp == SLOT_EMPTY) return false;
        if (sp != SLOT_TOMBSTONE && sp >= psl && m->key_eq(slot_key(m, i), key)) {
            if (val_out) memcpy(val_out, slot_val(m, i), m->val_size);
            return true;
        }
        /* robin-hood: if current slot's psl < ours, key can't be here */
        if (sp != SLOT_TOMBSTONE && sp < psl) return false;
        i = (i + 1) & mask;
        if (++psl > 128) return false; /* safety: shouldn't happen at load<0.75 */
    }
}

void hg_map_set(hg_map_t *m, const void *key, const void *val) {
    /* grow if load > 0.75 */
    if (m->count * 4 >= m->cap * 3) {
        hg_map_rehash(m, m->cap * 2);
    }

    uint32_t h = m->key_hash(key, m->seed);
    int64_t  mask = m->cap - 1;
    int64_t  i = (int64_t)(h & (uint32_t)mask);

    /* we'll robin-hood insert; carry a (key,val,psl) tuple */
    uint8_t *carry_key = (uint8_t*)alloca(m->key_size);
    uint8_t *carry_val = (uint8_t*)alloca(m->val_size);
    memcpy(carry_key, key, m->key_size);
    memcpy(carry_val, val, m->val_size);
    uint8_t carry_psl = 1;

    for (;;) {
        uint8_t sp = slot_psl(m, i);
        if (sp == SLOT_EMPTY || sp == SLOT_TOMBSTONE) {
            /* empty slot: place here */
            slot_at(m, i)[0] = carry_psl;
            memcpy(slot_key(m, i), carry_key, m->key_size);
            memcpy(slot_val(m, i), carry_val, m->val_size);
            m->count++;
            return;
        }
        /* check for existing key */
        if (m->key_eq(slot_key(m, i), carry_key)) {
            /* update value */
            memcpy(slot_val(m, i), carry_val, m->val_size);
            return;
        }
        /* robin-hood: steal from the rich */
        if (sp < carry_psl) {
            /* swap carry with current slot */
            uint8_t tmp_psl = sp;
            sp = carry_psl; carry_psl = tmp_psl;
            /* swap key */
            uint8_t tmp_buf[256];
            if (m->key_size <= sizeof(tmp_buf)) {
                memcpy(tmp_buf,         slot_key(m,i), m->key_size);
                memcpy(slot_key(m,i),  carry_key,      m->key_size);
                memcpy(carry_key,       tmp_buf,        m->key_size);
            }
            /* swap val */
            if (m->val_size <= sizeof(tmp_buf)) {
                memcpy(tmp_buf,         slot_val(m,i), m->val_size);
                memcpy(slot_val(m,i),  carry_val,      m->val_size);
                memcpy(carry_val,       tmp_buf,        m->val_size);
            }
            slot_at(m, i)[0] = sp;
        }
        i = (i + 1) & (m->cap - 1);
        carry_psl++;
    }
}

void hg_map_delete(hg_map_t *m, const void *key) {
    if (m->count == 0) return;
    uint32_t h = m->key_hash(key, m->seed);
    int64_t  mask = m->cap - 1;
    int64_t  i = (int64_t)(h & (uint32_t)mask);
    uint8_t  psl = 1;

    for (;;) {
        uint8_t sp = slot_psl(m, i);
        if (sp == SLOT_EMPTY) return;
        if (sp != SLOT_TOMBSTONE && sp >= psl && m->key_eq(slot_key(m, i), key)) {
            /* backward shift deletion */
            m->count--;
            for (;;) {
                int64_t j = (i + 1) & mask;
                uint8_t np = slot_psl(m, j);
                if (np == SLOT_EMPTY || np == SLOT_TOMBSTONE || np == 1) {
                    slot_at(m, i)[0] = SLOT_EMPTY;
                    break;
                }
                /* shift j into i, decrement PSL */
                slot_at(m, i)[0] = np - 1;
                memcpy(slot_key(m, i), slot_key(m, j), m->key_size);
                memcpy(slot_val(m, i), slot_val(m, j), m->val_size);
                i = j;
            }
            return;
        }
        if (sp != SLOT_TOMBSTONE && sp < psl) return;
        i = (i + 1) & mask;
        if (++psl > 128) return;
    }
}

int64_t hg_map_len(hg_map_t *m) { return m ? m->count : 0; }

void hg_map_iter_init(hg_map_t *m, hg_map_iter_t *it) {
    it->m       = m;
    it->started = false;
    /* randomize starting position for iteration order */
    uint32_t r  = m->seed ^ (uint32_t)(uintptr_t)it;
    r ^= r << 13; r ^= r >> 17; r ^= r << 5; /* xorshift32 */
    it->start   = (int64_t)(r % (uint32_t)m->cap);
    it->pos     = it->start;
}

bool hg_map_iter_next(hg_map_iter_t *it, void *key_out, void *val_out) {
    hg_map_t *m = it->m;
    if (!m || m->count == 0) return false;

    for (;;) {
        if (it->started && it->pos == it->start) return false;
        it->started = true;
        int64_t i = it->pos;
        it->pos = (it->pos + 1) % m->cap;

        uint8_t sp = slot_psl(m, i);
        if (sp == SLOT_EMPTY || sp == SLOT_TOMBSTONE) continue;
        if (key_out) memcpy(key_out, slot_key(m, i), m->key_size);
        if (val_out) memcpy(val_out, slot_val(m, i), m->val_size);
        return true;
    }
}
