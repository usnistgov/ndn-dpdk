#ifndef NDN_DPDK_CONTAINER_PCCT_PCC_KEY_H
#define NDN_DPDK_CONTAINER_PCCT_PCC_KEY_H

/// \file

#include "../../ndn/name.h"

/** \brief Hash key stored in \p PccEntry.
 */
typedef struct PccKey
{
  uint8_t name[NAME_MAX_LENGTH];
  uint8_t fh[NAME_MAX_LENGTH];
} PccKey;

/** \brief Hash key for searching among \p PccEntry.
 */
typedef struct PccSearch
{
  LName name;
  LName fh;
} PccSearch;

/** \brief Determine if a \p PccKey matches a \p PccSearch.
 */
static bool
PccKey_MatchSearchKey(const PccKey* key, const PccSearch* search)
{
  assert(search->name.length <= sizeof(key->name));
  assert(search->fh.length <= sizeof(key->fh));
  return memcmp(key->name, search->name.value, search->name.length) == 0 &&
         memcmp(key->fh, search->fh.value, search->fh.length) == 0;
}

/** \brief Copy \p PccSearch into a \p PccKey.
 */
void PccKey_CopyFromSearch(PccKey* key, const PccSearch* search);

#endif // NDN_DPDK_CONTAINER_PCCT_PCC_KEY_H
