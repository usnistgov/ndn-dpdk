#ifndef NDNDPDK_NDNI_NAME_H
#define NDNDPDK_NDNI_NAME_H

/** @file */

#include "common.h"

enum
{
  NameHexBufferLength = 2 * NameMaxLength + 1,
};

extern uint64_t LName_EmptyHash_;

/** @brief Name in linear buffer. */
typedef struct LName
{
  const uint8_t* value;
  uint16_t length;
} LName;

/**
 * @brief Determine whether @p a is a prefix of @b b .
 * @retval 0 @p a equals @p b .
 * @retval positive @p a is a prefix of @p b .
 * @retval negative otherwise.
 */
static inline int
LName_IsPrefix(LName a, LName b)
{
  if (a.length > b.length || memcmp(a.value, b.value, a.length) != 0) {
    return -1;
  }
  return b.length - a.length;
}

/** @brief Compute hash for a name. */
uint64_t
LName_ComputeHash(LName name);

/**
 * @brief Print name in hexadecimal format.
 * @return number of characters written, excluding trailing null character.
 */
__attribute__((nonnull)) int
LName_PrintHex(LName name, char buffer[NameHexBufferLength]);

/**
 * @brief Find a matching prefix of @p name .
 * @param name a packet name.
 * @param maxPrefix exclusive upper bound of @p prefixL vector.
 * @param prefixL a vector of name prefix TLV-LENGTH; zero indicates end of vector.
 * @param prefixV a buffer of name prefix TLV-VALUE, written consecutively.
 * @pre SUM(prefixL) <= cap(prefixV)
 * @return index of first matching prefix.
 * @retval -1 no matching prefix.
 */
__attribute__((nonnull)) static inline int
LNamePrefixFilter_Find(LName name, int maxPrefix, const uint16_t* prefixL, const uint8_t* prefixV)
{
  size_t offset = 0;
  for (int i = 0; i < maxPrefix; ++i) {
    if (prefixL[i] == 0) {
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
typedef struct PName
{
  const uint8_t* value; ///< TLV-VALUE
  uint16_t length;      ///< TLV-LENGTH
  uint16_t nComps;      ///< number of components
  bool hasDigestComp;   ///< ends with digest component?

  bool hasHashes_;                       ///< are hash[i] computed?
  uint16_t comp_[PNameCachedComponents]; ///< end offset of i-th component
  uint64_t hash_[PNameCachedComponents]; ///< hash of i+1-component prefix
} PName;

/** @brief Convert PName to LName. */
static __rte_always_inline LName
PName_ToLName(const PName* p)
{
  return *(const LName*)p;
}

/** @brief Parse a name from memory buffer. */
__attribute__((nonnull)) bool
PName_Parse(PName* p, LName l);

__attribute__((nonnull)) LName
PName_GetPrefix_Uncached_(const PName* p, int n);

/**
 * @brief Get a prefix of first @p n components.
 * @param n number of components; if negative, count from end.
 */
__attribute__((nonnull)) static inline LName
PName_GetPrefix(const PName* p, int n)
{
  if (n < 0) {
    n += p->nComps;
  }
  n = RTE_MIN(n, (int)p->nComps);

  if (unlikely(n <= 0)) {
    return (LName){ 0 };
  }

  if (unlikely(n > PNameCachedComponents)) {
    return PName_GetPrefix_Uncached_(p, n);
  }

  return (LName){ .length = p->comp_[n - 1], .value = p->value };
}

__attribute__((nonnull)) void
PName_PrepareHashes_(PName* p);

/**
 * @brief Compute hash for first @p i components.
 * @param i prefix length, must be no greater than n->nComps.
 */
__attribute__((nonnull)) static inline uint64_t
PName_ComputePrefixHash(const PName* p, uint16_t i)
{
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
PName_ComputeHash(const PName* p)
{
  return PName_ComputePrefixHash(p, p->nComps);
}

#endif // NDNDPDK_NDNI_NAME_H
