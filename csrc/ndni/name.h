#ifndef NDNDPDK_NDNI_NAME_H
#define NDNDPDK_NDNI_NAME_H

/** @file */

#include "common.h"

extern uint64_t LName_EmptyHash_;

/** @brief Name in linear buffer. */
typedef struct LName {
  const uint8_t* value;
  uint16_t length;
} LName;

__attribute__((nonnull)) static __rte_always_inline bool
LName_ParseVarNum_(LName name, uint16_t* restrict pos, uint16_t* restrict n, uint16_t minTail) {
  if (unlikely(*pos + 1 + minTail > name.length)) {
    return false;
  }

  *n = name.value[*pos];
  *pos += 1;
  if (likely(*n < 0xFD)) {
    return true;
  }

  if (unlikely(*n > 0xFD) || unlikely(*pos + 2 + minTail > name.length)) {
    return false;
  }

  *n = rte_be_to_cpu_16(*(unaligned_uint16_t*)&name.value[*pos]);
  *pos += 2;
  return true;
}

/**
 * @brief Iterate over name components.
 * @code
 * uint16_t pos = 0, type = 0, length = 0;
 * while (likely(LName_Component(name, &pos, &type, &length))) {
 *   uint8_t* value = &name.value[pos];
 *   pos += length;
 * }
 * @endcode
 */
__attribute__((nonnull)) static inline bool
LName_Component(LName name, uint16_t* restrict pos, uint16_t* restrict type,
                uint16_t* restrict length) {
  return LName_ParseVarNum_(name, pos, type, 1) && likely(*type != 0) &&
         LName_ParseVarNum_(name, pos, length, 0) && *pos + *length <= name.length;
}

/** @brief Determine whether @p a equals @p b . */
static inline bool
LName_Equal(LName a, LName b) {
  return a.length == b.length && memcmp(a.value, b.value, a.length) == 0;
}

/**
 * @brief Determine whether @p a is a prefix of @p b .
 * @retval 0 @p a equals @p b .
 * @retval positive @p a is a prefix of @p b .
 * @retval negative otherwise.
 */
static inline int
LName_IsPrefix(LName a, LName b) {
  if (a.length > b.length || memcmp(a.value, b.value, a.length) != 0) {
    return -1;
  }
  return b.length - a.length;
}

static __rte_always_inline LName
LName_SliceByte_(LName name, uint16_t start, uint16_t end) {
  return (LName){.length = end - start, .value = RTE_PTR_ADD(name.value, start)};
}

/**
 * @brief Get a sub name of @c [start:end) byte range.
 * @param start first byte offset (inclusive).
 * @param end last byte offset (exclusive).
 */
static __rte_always_inline LName
LName_SliceByte(LName name, uint16_t start, uint16_t end) {
  end = RTE_MIN(end, name.length);
  if (unlikely(start >= end)) {
    return (LName){0};
  }
  return LName_SliceByte_(name, start, end);
}

static __rte_always_inline LName
LName_Slice_(LName name, uint16_t start, uint16_t end) {
  uint16_t i = 0, pos = 0, type = 0, length = 0;
  uint16_t posStart = likely(start == 0) ? 0 : name.length;
  uint16_t posEnd = name.length;
  while (likely(LName_Component(name, &pos, &type, &length))) {
    ++i;
    pos += length;
    if (i == start) {
      posStart = pos;
    } else if (i == end) {
      posEnd = pos;
      break;
    }
  }
  return LName_SliceByte_(name, posStart, posEnd);
}

/**
 * @brief Get a sub name of @c [start:end) components.
 * @param start first component index (inclusive).
 * @param end last component index (exclusive).
 */
static inline LName
LName_Slice(LName name, uint16_t start, uint16_t end) {
  if (unlikely(start >= end)) {
    return (LName){0};
  }
  return LName_Slice_(name, start, end);
}

/** @brief Compute hash for a name. */
uint64_t
LName_ComputeHash(LName name);

/**
 * @brief Find a matching prefix of @p name .
 * @param name a packet name.
 * @param maxPrefix exclusive upper bound of @p prefixL vector.
 * @param prefixL a vector of name prefix TLV-LENGTH; UINT16_MAX indicates end of vector.
 * @param prefixV a buffer of name prefix TLV-VALUE, written consecutively.
 * @pre SUM(prefixL) <= cap(prefixV)
 * @return index of first matching prefix.
 * @retval -1 no matching prefix.
 */
__attribute__((nonnull)) static inline int
LNamePrefixFilter_Find(LName name, int maxPrefix, const uint16_t* prefixL, const uint8_t* prefixV) {
  size_t offset = 0;
  for (int i = 0; i < maxPrefix; ++i) {
    if (prefixL[i] == UINT16_MAX) {
      break;
    }

    LName prefix = {
      .value = RTE_PTR_ADD(prefixV, offset),
      .length = prefixL[i],
    };
    if (LName_IsPrefix(prefix, name) >= 0) {
      return i;
    }
    offset += prefix.length;
  }
  return -1;
}

/** @brief Parsed name. */
typedef struct PName {
  const uint8_t* value; ///< TLV-VALUE
  uint16_t length;      ///< TLV-LENGTH
  uint16_t nComps;      ///< number of components
  struct {
    int16_t firstNonGeneric : 12; ///< index of first non-generic component
    bool hasDigestComp : 1;       ///< ends with digest component?
    bool hasHashes_ : 1;          ///< hash_ computed?
    uint32_t a_ : 2;
  } __rte_packed;
  uint16_t comp_[PNameCachedComponents]; ///< end offset of i-th component
  uint64_t hash_[PNameCachedComponents]; ///< hash of i+1-component prefix
} PName;
// maximum component index must fit in firstNonGeneric
static_assert((NameMaxLength / 2) <= (1 << 11), "");

/** @brief Convert PName to LName. */
static __rte_always_inline LName
PName_ToLName(const PName* p) {
  static_assert(offsetof(LName, value) == offsetof(PName, value), "");
  static_assert(offsetof(LName, length) == offsetof(PName, length), "");
  return *(const LName*)p;
}

/** @brief Parse a name from memory buffer. */
__attribute__((nonnull)) bool
PName_Parse(PName* p, LName l);

__attribute__((nonnull)) static __rte_noinline LName
PName_Slice_Uncached_(const PName* p, int16_t start, int16_t end) {
  return LName_Slice_(PName_ToLName(p), (uint16_t)start, (uint16_t)end);
}

/**
 * @brief Get a sub name of @c [start:end-1] components.
 * @param start first component index (inclusive); if negative, count from end.
 * @param end last component index (exclusive); if negative, count from end.
 */
__attribute__((nonnull)) static __rte_always_inline LName
PName_Slice(const PName* p, int16_t start, int16_t end) {
  if (unlikely(start < 0)) {
    start += p->nComps;
  }
  start = CLAMP(start, 0, (int16_t)p->nComps);

  if (unlikely(end < 0)) {
    end += p->nComps;
  }
  end = CLAMP(end, 0, (int16_t)p->nComps);

  if (unlikely(start >= end)) {
    return (LName){0};
  }

  if (unlikely(end > PNameCachedComponents)) {
    return PName_Slice_Uncached_(p, start, end);
  }

  return LName_SliceByte_(PName_ToLName(p), likely(start == 0) ? 0 : p->comp_[start - 1],
                          p->comp_[end - 1]);
}

/**
 * @brief Get a prefix of first @p n components.
 * @param n number of components; if negative, count from end.
 */
__attribute__((nonnull)) static __rte_always_inline LName
PName_GetPrefix(const PName* p, int16_t n) {
  return PName_Slice(p, 0, n);
}

__attribute__((nonnull)) void
PName_PrepareHashes_(PName* p);

/**
 * @brief Compute hash for first @p i components.
 * @param i prefix length, must be no greater than n->nComps.
 */
__attribute__((nonnull)) static inline uint64_t
PName_ComputePrefixHash(const PName* p, uint16_t i) {
  NDNDPDK_ASSERT(i <= p->nComps);
  if (unlikely(i == 0)) {
    return LName_EmptyHash_;
  }
  if (unlikely(i > PNameCachedComponents)) {
    return LName_ComputeHash(PName_GetPrefix(p, i));
  }

  if (!p->hasHashes_) {
    PName_PrepareHashes_((PName*)p);
  }
  return p->hash_[i - 1];
}

/** @brief Compute hash for the name. */
__attribute__((nonnull)) static inline uint64_t
PName_ComputeHash(const PName* p) {
  return PName_ComputePrefixHash(p, p->nComps);
}

#endif // NDNDPDK_NDNI_NAME_H
