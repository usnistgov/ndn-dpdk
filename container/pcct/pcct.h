#ifndef NDN_DPDK_CONTAINER_PCI_PCI_H
#define NDN_DPDK_CONTAINER_PCI_PCI_H

/// \file

#include <rte_hash.h>

#include "cs-struct.h"
#include "pcc-entry.h"
#include "pit-struct.h"

/** \brief Shared index for PIT and CS.
 *
 *  \p PcctPriv is attached to the private data area of this mempool.
 */
typedef struct rte_mempool Pcct;

/** \brief rte_mempool private data for Pcc.
 */
typedef struct PcctPriv
{
  PccEntry* keyHt;
  struct rte_hash* tokenHt;
  uint64_t lastToken;

  PitPriv pitPriv;
  CsPriv csPriv;
} PcctPriv;

#define Pcct_GetPriv(pcct)                                                     \
  ((PcctPriv*)rte_mempool_get_priv((struct rte_mempool*)(pcct)))

/** \brief Create a PIT-CS index.
 *  \param id identifier for debugging, must be unique.
 *  \param maxEntries maximum number of entries, should be (2^q-1).
 *  \param numaSocket where to allocate memory.
 */
Pcct* Pcct_New(const char* id, uint32_t maxEntries, unsigned numaSocket);

/** \brief Release all memory.
 */
void Pcct_Close(Pcct* pcct);

/** \brief Insert or find an entry.
 *  \param[out] isNew whether the entry is new
 */
PccEntry* Pcct_Insert(Pcct* pcct, uint64_t hash, PccSearch* search,
                      bool* isNew);

/** \brief Erase an entry.
 */
void Pcct_Erase(Pcct* pcct, PccEntry* entry);

/** \brief Find an entry.
 */
PccEntry* Pcct_Find(const Pcct* pcct, uint64_t hash, PccSearch* search);

/** \brief Assign a token to an entry.
 */
void Pcct_AddToken(Pcct* pcct, PccEntry* entry);

/** \brief Clear the token on an entry.
 */
void Pcct_RemoveToken(Pcct* pcct, PccEntry* entry);

/** \brief Find an entry by token.
 */
PccEntry* Pcct_FindByToken(const Pcct* pcct, uint64_t token);

#endif // NDN_DPDK_CONTAINER_PCI_PCI_H
