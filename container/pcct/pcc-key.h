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

/** \brief Hash key stored in \c PccEntry.
 */
typedef struct PccKey
{
  uint8_t nameV[NAME_MAX_LENGTH];
  uint8_t fhV[NAME_MAX_LENGTH];
  uint16_t nameL;
  uint16_t fhL;
} PccKey;

/** \brief Determine if \p key->name matches \p name.
 */
static bool
PccKey_MatchName(const PccKey* key, LName name)
{
  return name.length == key->nameL &&
         memcmp(key->nameV, name.value, key->nameL) == 0;
}

/** \brief Determine if \p key matches \p search.
 */
static bool
PccKey_MatchSearchKey(const PccKey* key, const PccSearch* search)
{
  return search->fh.length == key->fhL && PccKey_MatchName(key, search->name) &&
         memcmp(key->fhV, search->fh.value, search->fh.length) == 0;
}

/** \brief Copy \c search into \p key.
 */
void PccKey_CopyFromSearch(PccKey* key, const PccSearch* search);

#endif // NDN_DPDK_CONTAINER_PCCT_PCC_KEY_H
