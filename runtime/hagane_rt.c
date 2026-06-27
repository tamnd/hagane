/*
 * hagane_rt.c — C11 runtime for the hagane Go→C transpiler
 *
 * Implements the core ABI: allocation, strings, slices, and init.
 */

#include "hagane_rt.h"

#include <stdlib.h>
#include <string.h>
#include <stdint.h>
#include <stdio.h>
#include <time.h>
#include <unistd.h>

/* -------------------------------------------------------------------------
 * Allocation
 * ---------------------------------------------------------------------- */

void *hg_alloc(size_t size) {
    if (size == 0) {
        size = 1;
    }
    void *p = calloc(1, size);
    if (p == NULL) {
        fputs("hagane: out of memory\n", stderr);
        abort();
    }
    return p;
}

void *hg_realloc(void *ptr, size_t old_size, size_t new_size) {
    void *p = realloc(ptr, new_size);
    if (p == NULL) {
        fputs("hagane: out of memory\n", stderr);
        abort();
    }
    /* Zero the newly added bytes if the allocation grew. */
    if (new_size > old_size) {
        memset((char *)p + old_size, 0, new_size - old_size);
    }
    return p;
}

/* -------------------------------------------------------------------------
 * Strings
 *
 * hg_string_t holds a (data, len) pair; the backing bytes are owned by the
 * string and allocated via hg_alloc.  Strings are immutable after creation.
 * ---------------------------------------------------------------------- */

hg_string_t hg_string_concat(hg_string_t a, hg_string_t b) {
    int64_t total = a.len + b.len;
    /* Handle the empty result without a zero-sized allocation. */
    if (total == 0) {
        hg_string_t s = { .data = NULL, .len = 0 };
        return s;
    }
    char *buf = (char *)hg_alloc((size_t)total);
    if (a.len > 0) {
        memcpy(buf, a.data, (size_t)a.len);
    }
    if (b.len > 0) {
        memcpy(buf + a.len, b.data, (size_t)b.len);
    }
    hg_string_t s = { .data = buf, .len = total };
    return s;
}

bool hg_string_equal(hg_string_t a, hg_string_t b) {
    if (a.len != b.len) {
        return false;
    }
    if (a.len == 0) {
        return true;
    }
    return memcmp(a.data, b.data, (size_t)a.len) == 0;
}

int hg_string_compare(hg_string_t a, hg_string_t b) {
    int64_t min_len = a.len < b.len ? a.len : b.len;
    if (min_len > 0) {
        int r = memcmp(a.data, b.data, (size_t)min_len);
        if (r != 0) {
            return r;
        }
    }
    /* Shorter string is less; equal lengths means equal strings. */
    if (a.len < b.len) return -1;
    if (a.len > b.len) return  1;
    return 0;
}

/* -------------------------------------------------------------------------
 * Slices (raw / element-size-generic helpers)
 *
 * The transpiler emits calls to these helpers with the concrete elem_size
 * known at compile time.  The slice header (ptr, len, cap) is managed by
 * the generated C code; only the backing array is managed here.
 * ---------------------------------------------------------------------- */

void *hg_makeslice_raw(size_t elem_size, int64_t len, int64_t cap) {
    if (len < 0) {
        fputs("hagane: makeslice: len out of range\n", stderr);
        abort();
    }
    if (cap < len) {
        fputs("hagane: makeslice: cap out of range\n", stderr);
        abort();
    }
    /* Overflow check: cap * elem_size must fit in size_t. */
    if (elem_size > 0 && (uint64_t)cap > (SIZE_MAX / elem_size)) {
        fputs("hagane: makeslice: len out of range\n", stderr);
        abort();
    }
    if (cap == 0) {
        /* Return a non-NULL sentinel so ptr != NULL means "allocated". */
        return hg_alloc(1);
    }
    void *p = calloc((size_t)cap, elem_size);
    if (p == NULL) {
        fputs("hagane: out of memory\n", stderr);
        abort();
    }
    return p;
}

void *hg_growslice_raw(size_t elem_size, void *old, int64_t len,
                       int64_t *cap_out, int64_t extra) {
    int64_t new_cap = *cap_out * 2;
    if (new_cap < len + extra) {
        new_cap = len + extra;
    }
    if (new_cap < 4) {
        new_cap = 4;
    }
    void *p = realloc(old, (size_t)new_cap * elem_size);
    if (p == NULL) {
        fputs("hagane: out of memory\n", stderr);
        abort();
    }
    /* Zero the freshly added capacity. */
    if (new_cap > *cap_out) {
        memset((char *)p + *cap_out * elem_size, 0,
               (size_t)(new_cap - *cap_out) * elem_size);
    }
    *cap_out = new_cap;
    return p;
}

/* -------------------------------------------------------------------------
 * Memory move
 * ---------------------------------------------------------------------- */

void hg_memmove(void *dst, const void *src, size_t n) {
    memmove(dst, src, n);
}

/* -------------------------------------------------------------------------
 * Runtime initialisation
 * ---------------------------------------------------------------------- */

void hg_runtime_init(void) {
    /* Seed the PRNG used for randomised map iteration order.
     * XOR with pid so two processes started in the same second differ. */
    srand((unsigned)(time(NULL) ^ (uintptr_t)getpid()));

    /* Future: scheduler / goroutine pool init goes here. */
}
