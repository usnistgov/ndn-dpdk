#ifndef NDN_DPDK_CONTAINER_PCCT_PCC_KEY_H
#define NDN_DPDK_CONTAINER_PCCT_PCC_KEY_H

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

/** \brief Compute hash value for use in PCCT.
 */
static uint64_t
PccSearch_ComputeHash(const PccSearch* search)
{
  return search->nameHash ^ search->fhHash;
}

/** \brief Convert \p search to a string for debug purpose.
 *  \return A string from thread-local buffer.
 *  \warning Subsequent *ToDebugString calls on the same thread overwrite the buffer.
 */
const char* PccSearch_ToDebugString(const PccSearch* search);

#define PCC_KEY_NAME_CAP 240
#define PCC_KEY_FH_CAP 160
#define PCC_KEY_EXT_CAP 1000

typedef struct PccKeyExt PccKeyExt;

/** \brief Hash key stored in \c PccEntry.
 */
typedef struct PccKey
{
  uint8_t nameV[PCC_KEY_NAME_CAP];
  uint8_t fhV[PCC_KEY_FH_CAP];
  PccKeyExt* nameExt;
  PccKeyExt* fhExt;
  uint16_t nameL;
  uint16_t fhL;
} PccKey;

struct PccKeyExt
{
  PccKeyExt* next;
  uint8_t value[PCC_KEY_EXT_CAP];
};

static bool
__PccKey_MatchNameOrFhV(LName name, const uint8_t* value, uint16_t cap,
                        const PccKeyExt* ext)
{
  if (memcmp(value, name.value, RTE_MIN(name.length, cap)) != 0) {
    return false;
  }
  for (uint16_t offset = cap; unlikely(offset < name.length);
       offset += PCC_KEY_EXT_CAP) {
    assert(ext != NULL);
    if (memcmp(ext->value, RTE_PTR_ADD(name.value, offset),
               RTE_MIN(name.length - offset, PCC_KEY_EXT_CAP)) != 0) {
      return false;
    }
    ext = ext->next;
  }
  return true;
}

/** \brief Determine if \p key->name equals \p name.
 */
static bool
PccKey_MatchName(const PccKey* key, LName name)
{
  return name.length == key->nameL &&
         __PccKey_MatchNameOrFhV(name, key->nameV, PCC_KEY_NAME_CAP,
                                 key->nameExt);
}

/** \brief Determine if \p key matches \p search.
 */
static bool
PccKey_MatchSearchKey(const PccKey* key, const PccSearch* search)
{
  return search->fh.length == key->fhL && PccKey_MatchName(key, search->name) &&
         __PccKey_MatchNameOrFhV(search->fh, key->fhV, PCC_KEY_FH_CAP,
                                 key->fhExt);
}

#define __PccKey_CountExtensionsOn(excess)                                     \
  ((excess) / PCC_KEY_EXT_CAP + (bool)((excess) % PCC_KEY_EXT_CAP > 0))

#define __PccKey_CountExtensions(nameL, fhL)                                   \
  (__PccKey_CountExtensionsOn(nameL - PCC_KEY_NAME_CAP) +                      \
   __PccKey_CountExtensionsOn(fhL - PCC_KEY_FH_CAP))

/** \brief Determine how many PccKeyExts are needed to copy \p search into PccKey.
 */
static int
PccKey_CountExtensions(const PccSearch* search)
{
  return __PccKey_CountExtensions(search->name.length, search->fh.length);
}

/** \brief Maximum return value of PccKey_CountExtensions.
 */
#define PCC_KEY_MAX_EXTS                                                       \
  __PccKey_CountExtensions(NAME_MAX_LENGTH, NAME_MAX_LENGTH)

static int
__PccKey_CopyNameOrFhV(LName name, uint8_t* value, uint16_t cap,
                       PccKeyExt** next, PccKeyExt* exts[])
{
  rte_memcpy(value, name.value, RTE_MIN(name.length, cap));
  int nExts = 0;
  for (uint16_t offset = cap; unlikely(offset < name.length);
       offset += PCC_KEY_EXT_CAP) {
    PccKeyExt* ext = exts[nExts++];
    *next = ext;
    rte_memcpy(ext->value, RTE_PTR_ADD(name.value, offset),
               RTE_MIN(name.length - offset, PCC_KEY_EXT_CAP));
    next = &ext->next;
  }
  return nExts;
}

/** \brief Copy \c search into \p key.
 */
static void
PccKey_CopyFromSearch(PccKey* key, const PccSearch* search, PccKeyExt* exts[],
                      int nExts)
{
  assert(nExts == PccKey_CountExtensions(search));
  key->nameL = search->name.length;
  key->fhL = search->fh.length;
  int nNameExts = __PccKey_CopyNameOrFhV(search->name, key->nameV,
                                         PCC_KEY_NAME_CAP, &key->nameExt, exts);
  __PccKey_CopyNameOrFhV(search->fh, key->fhV, PCC_KEY_FH_CAP, &key->fhExt,
                         &exts[nNameExts]);
}

/** \brief Move PccKeyExts into \p exts to prepare for removal.
 */
static int
PccKey_StripExts(PccKey* key, PccKeyExt* exts[PCC_KEY_MAX_EXTS])
{
  int nExts = 0;
  for (PccKeyExt* ext = key->nameExt; unlikely(ext != NULL); ext = ext->next) {
    exts[nExts++] = ext;
  }
  for (PccKeyExt* ext = key->fhExt; unlikely(ext != NULL); ext = ext->next) {
    exts[nExts++] = ext;
  }
  assert(nExts == __PccKey_CountExtensions(key->nameL, key->fhL));
  return nExts;
}

#endif // NDN_DPDK_CONTAINER_PCCT_PCC_KEY_H
