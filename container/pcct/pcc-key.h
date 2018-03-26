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
} PccSearch;

/** \brief Convert \p search to a string for debug purpose.
 *  \return A string from thread-local buffer.
 *  \warning Subsequent *ToDebugString calls on the same thread overwrite the buffer.
 */
const char* PccSearch_ToDebugString(const PccSearch* search);

/** \brief Hash key stored in \c PccEntry.
 */
typedef struct PccKey
{
  uint8_t name[NAME_MAX_LENGTH];
  uint8_t fh[NAME_MAX_LENGTH];
} PccKey;

/** \brief Determine if \p key matches \p search.
 */
static bool
PccKey_MatchSearchKey(const PccKey* key, const PccSearch* search)
{
  assert(search->name.length <= sizeof(key->name));
  assert(search->fh.length <= sizeof(key->fh));
  return memcmp(key->name, search->name.value, search->name.length) == 0 &&
         memcmp(key->fh, search->fh.value, search->fh.length) == 0;
}

/** \brief Copy \c search into \p key.
 */
void PccKey_CopyFromSearch(PccKey* key, const PccSearch* search);

#endif // NDN_DPDK_CONTAINER_PCCT_PCC_KEY_H
