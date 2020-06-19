#ifndef NDN_DPDK_PCCT_PCC_KEY_H
#define NDN_DPDK_PCCT_PCC_KEY_H

/// \file

#include "common.h"

/** \brief Hash key for searching among \c PccEntry.
 */
typedef struct PccSearch
{
  LName name;
  LName fh;
  uint64_t nameHash;
  uint64_t fhHash;
} PccSearch;

/** \brief Initialize PccSearch from name and Interest fwhint.
 */
static inline void
PccSearch_FromNames(PccSearch* search,
                    const Name* name,
                    const PInterest* interest)
{
  const LName* lname = (const LName*)name;
  search->name = *lname;
  search->nameHash = PName_ComputeHash(&name->p, name->v);
  if (interest->activeFh >= 0) {
    const LName* fhLName = (const LName*)&interest->activeFhName;
    search->fh = *fhLName;
    search->fhHash =
      PName_ComputeHash(&interest->activeFhName.p, interest->activeFhName.v);
  } else {
    search->fh.length = 0;
    search->fhHash = 0;
  }
}

/** \brief Compute hash value for use in PCCT.
 */
static inline uint64_t
PccSearch_ComputeHash(const PccSearch* search)
{
  return search->nameHash ^ search->fhHash;
}

/** \brief Convert \p search to a string for debug purpose.
 *  \return A string from thread-local buffer.
 *  \warning Subsequent *ToDebugString calls on the same thread overwrite the buffer.
 */
const char*
PccSearch_ToDebugString(const PccSearch* search);

#define PCC_KEY_NAME_CAP 240
#define PCC_KEY_FH_CAP 160
#define PCC_KEY_EXT_CAP 1000

typedef struct PccKeyExt PccKeyExt;

/** \brief Hash key stored in \c PccEntry.
 */
typedef struct PccKey
{
  PccKeyExt* nameExt;
  PccKeyExt* fhExt;
  uint16_t nameL;
  uint16_t fhL;
  uint8_t nameV[PCC_KEY_NAME_CAP];
  uint8_t fhV[PCC_KEY_FH_CAP];
} PccKey;

struct PccKeyExt
{
  PccKeyExt* next;
  uint8_t value[PCC_KEY_EXT_CAP];
};

static inline bool
PccKey_MatchNameOrFhV_(LName name,
                       const uint8_t* value,
                       uint16_t cap,
                       const PccKeyExt* ext)
{
  if (memcmp(value, name.value, RTE_MIN(name.length, cap)) != 0) {
    return false;
  }
  for (uint16_t offset = cap; unlikely(offset < name.length);
       offset += PCC_KEY_EXT_CAP) {
    assert(ext != NULL);
    if (memcmp(ext->value,
               RTE_PTR_ADD(name.value, offset),
               RTE_MIN(name.length - offset, PCC_KEY_EXT_CAP)) != 0) {
      return false;
    }
    ext = ext->next;
  }
  return true;
}

/** \brief Determine if \p key->name equals \p name.
 */
static inline bool
PccKey_MatchName(const PccKey* key, LName name)
{
  return name.length == key->nameL &&
         PccKey_MatchNameOrFhV_(
           name, key->nameV, PCC_KEY_NAME_CAP, key->nameExt);
}

/** \brief Determine if \p key matches \p search.
 */
static inline bool
PccKey_MatchSearchKey(const PccKey* key, const PccSearch* search)
{
  return search->fh.length == key->fhL && PccKey_MatchName(key, search->name) &&
         PccKey_MatchNameOrFhV_(
           search->fh, key->fhV, PCC_KEY_FH_CAP, key->fhExt);
}

#define PccKey_CountExtensionsOn_(excess)                                      \
  ((excess) / PCC_KEY_EXT_CAP + (bool)((excess) % PCC_KEY_EXT_CAP > 0))

#define PccKey_CountExtensions_(nameL, fhL)                                    \
  (PccKey_CountExtensionsOn_(nameL - PCC_KEY_NAME_CAP) +                       \
   PccKey_CountExtensionsOn_(fhL - PCC_KEY_FH_CAP))

/** \brief Determine how many PccKeyExts are needed to copy \p search into PccKey.
 */
static inline int
PccKey_CountExtensions(const PccSearch* search)
{
  return PccKey_CountExtensions_(search->name.length, search->fh.length);
}

/** \brief Maximum return value of PccKey_CountExtensions.
 */
#define PCC_KEY_MAX_EXTS PccKey_CountExtensions_(NameMaxLength, NameMaxLength)

static inline int
PccKey_CopyNameOrFhV_(LName name,
                      uint8_t* value,
                      uint16_t cap,
                      PccKeyExt** next,
                      PccKeyExt* exts[])
{
  rte_memcpy(value, name.value, RTE_MIN(name.length, cap));
  int nExts = 0;
  for (uint16_t offset = cap; unlikely(offset < name.length);
       offset += PCC_KEY_EXT_CAP) {
    PccKeyExt* ext = exts[nExts++];
    *next = ext;
    rte_memcpy(ext->value,
               RTE_PTR_ADD(name.value, offset),
               RTE_MIN(name.length - offset, PCC_KEY_EXT_CAP));
    next = &ext->next;
  }
  *next = NULL;
  return nExts;
}

/** \brief Copy \c search into \p key.
 */
static inline void
PccKey_CopyFromSearch(PccKey* key,
                      const PccSearch* search,
                      PccKeyExt* exts[],
                      int nExts)
{
  assert(nExts == PccKey_CountExtensions(search));
  key->nameL = search->name.length;
  key->fhL = search->fh.length;
  int nNameExts = PccKey_CopyNameOrFhV_(
    search->name, key->nameV, PCC_KEY_NAME_CAP, &key->nameExt, exts);
  PccKey_CopyNameOrFhV_(
    search->fh, key->fhV, PCC_KEY_FH_CAP, &key->fhExt, &exts[nNameExts]);
}

/** \brief Move PccKeyExts into \p exts to prepare for removal.
 */
static inline int
PccKey_StripExts(PccKey* key, PccKeyExt* exts[PCC_KEY_MAX_EXTS])
{
  int nExts = 0;
  for (PccKeyExt* ext = key->nameExt; unlikely(ext != NULL); ext = ext->next) {
    exts[nExts++] = ext;
  }
  for (PccKeyExt* ext = key->fhExt; unlikely(ext != NULL); ext = ext->next) {
    exts[nExts++] = ext;
  }
  assert(nExts == PccKey_CountExtensions_(key->nameL, key->fhL));
  return nExts;
}

#endif // NDN_DPDK_PCCT_PCC_KEY_H
