#ifndef NDNDPDK_PCCT_PCC_KEY_H
#define NDNDPDK_PCCT_PCC_KEY_H

/** @file */

#include "common.h"

#define PccKey_CountExtensions_(nameL, fhL)                                                        \
  (DIV_CEIL(nameL - PccKeyNameCapacity, PccKeyExtCapacity) +                                       \
   DIV_CEIL(fhL - PccKeyFhCapacity, PccKeyExtCapacity))

enum
{
  PccKeyNameCapacity = 240,
  PccKeyFhCapacity = 160,
  PccKeyExtCapacity = 1000,
  PccKeyMaxExts = PccKey_CountExtensions_(NameMaxLength, NameMaxLength),
};

/** @brief Hash key for searching among @c PccEntry . */
typedef struct PccSearch
{
  LName name;
  LName fh;
  uint64_t nameHash;
  uint64_t fhHash;
} PccSearch;

/** @brief Initialize PccSearch from name and Interest fwhint. */
__attribute__((nonnull)) static inline void
PccSearch_FromNames(PccSearch* search, const PName* name, const PInterest* interest)
{
  search->name = PName_ToLName(name);
  search->nameHash = PName_ComputeHash(name);
  if (interest->activeFwHint >= 0) {
    search->fh = PName_ToLName(&interest->fwHint);
    search->fhHash = PName_ComputeHash(&interest->fwHint);
  } else {
    search->fh.length = 0;
    search->fhHash = 0;
  }
}

/** @brief Compute hash value for use in PCCT. */
__attribute__((nonnull)) static __rte_always_inline uint64_t
PccSearch_ComputeHash(const PccSearch* search)
{
  return search->nameHash ^ search->fhHash;
}

/**
 * @brief Convert @p search to a string for debug purpose.
 * @return pointer to a per-lcore static buffer that will be overwritten on subsequent calls.
 */
__attribute__((nonnull, returns_nonnull)) const char*
PccSearch_ToDebugString(const PccSearch* search);

typedef struct PccKeyExt PccKeyExt;

/** @brief Hash key stored in @c PccEntry . */
typedef struct PccKey
{
  PccKeyExt* nameExt;
  PccKeyExt* fhExt;
  uint16_t nameL;
  uint16_t fhL;
  uint8_t nameV[PccKeyNameCapacity];
  uint8_t fhV[PccKeyFhCapacity];
} PccKey;

struct PccKeyExt
{
  PccKeyExt* next;
  uint8_t value[PccKeyExtCapacity];
};

__attribute__((nonnull)) bool
PccKey_MatchFieldWithExt_(LName name, const uint8_t* firstV, uint16_t firstCapacity,
                          const PccKeyExt* ext);

__attribute__((nonnull(2))) static __rte_always_inline bool
PccKey_MatchField_(LName name, const uint8_t* firstV, uint16_t firstCapacity, const PccKeyExt* ext)
{
  if (unlikely(name.length > firstCapacity)) {
    return PccKey_MatchFieldWithExt_(name, firstV, firstCapacity, ext);
  }
  return memcmp(firstV, name.value, name.length) == 0;
}

/** @brief Determine if @c key->name equals @p name . */
__attribute__((nonnull)) static inline bool
PccKey_MatchName(const PccKey* key, LName name)
{
  return name.length == key->nameL &&
         PccKey_MatchField_(name, key->nameV, PccKeyNameCapacity, key->nameExt);
}

/** @brief Determine if @p key matches @p search . */
__attribute__((nonnull)) static inline bool
PccKey_MatchSearch(const PccKey* key, const PccSearch* search)
{
  return search->fh.length == key->fhL && PccKey_MatchName(key, search->name) &&
         PccKey_MatchField_(search->fh, key->fhV, PccKeyFhCapacity, key->fhExt);
}

/** @brief Determine how many PccKeyExts are needed to copy @p search into PccKey. */
__attribute__((nonnull)) static inline int
PccKey_CountExtensions(const PccSearch* search)
{
  return PccKey_CountExtensions_(search->name.length, search->fh.length);
}

__attribute__((nonnull)) int
PccKey_WriteFieldWithExt_(LName name, uint8_t* firstV, uint16_t firstCapacity, PccKeyExt** next,
                          PccKeyExt* exts[]);

__attribute__((nonnull)) static __rte_always_inline int
PccKey_WriteField_(LName name, uint8_t* firstV, uint16_t firstCapacity, PccKeyExt** next,
                   PccKeyExt* exts[])
{
  if (unlikely(name.length > firstCapacity)) {
    return PccKey_WriteFieldWithExt_(name, firstV, firstCapacity, next, exts);
  }
  rte_memcpy(firstV, name.value, name.length);
  *next = NULL;
  return 0;
}

/** @brief Copy @c search into @p key . */
__attribute__((nonnull)) static inline void
PccKey_CopyFromSearch(PccKey* key, const PccSearch* search, PccKeyExt* exts[], int nExts)
{
  NDNDPDK_ASSERT(nExts == PccKey_CountExtensions(search));
  key->nameL = search->name.length;
  key->fhL = search->fh.length;
  int nNameExts =
    PccKey_WriteField_(search->name, key->nameV, PccKeyNameCapacity, &key->nameExt, exts);
  PccKey_WriteField_(search->fh, key->fhV, PccKeyFhCapacity, &key->fhExt, &exts[nNameExts]);
}

/** @brief Move PccKeyExts into @p exts to prepare for removal. */
__attribute__((nonnull)) static inline int
PccKey_StripExts(PccKey* key, PccKeyExt* exts[PccKeyMaxExts])
{
  int nExts = 0;
  for (PccKeyExt* ext = key->nameExt; unlikely(ext != NULL); ext = ext->next) {
    exts[nExts++] = ext;
  }
  for (PccKeyExt* ext = key->fhExt; unlikely(ext != NULL); ext = ext->next) {
    exts[nExts++] = ext;
  }
  NDNDPDK_ASSERT(nExts == PccKey_CountExtensions_(key->nameL, key->fhL));
  return nExts;
}

#endif // NDNDPDK_PCCT_PCC_KEY_H
