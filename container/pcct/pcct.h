#ifndef NDN_DPDK_CONTAINER_PCCT_PCCT_H
#define NDN_DPDK_CONTAINER_PCCT_PCCT_H

/// \file

#include "cs-struct.h"
#include "pcc-entry.h"
#include "pit-struct.h"

/** \brief The PIT-CS Composite Table (PCCT).
 *
 *  Pcct* is struct rte_mempool* with \p PcctPriv attached to its private data area.
 */
typedef struct Pcct
{
} Pcct;

/** \brief Cast Pcct* as rte_mempool*.
 */
static inline struct rte_mempool*
Pcct_ToMempool(const Pcct* pcct)
{
  return (struct rte_mempool*)pcct;
}

/** \brief rte_mempool private data for Pcc.
 */
typedef struct PcctPriv
{
  PccEntry* keyHt;
  struct rte_hash* tokenHt;
  uint64_t lastToken;

  PitPriv pitPriv;
  CsPriv csPriv;

  uint32_t nKeyHtBuckets;
} PcctPriv;

/** \brief Access PcctPriv* struct.
 */
static inline PcctPriv*
Pcct_GetPriv(const Pcct* pcct)
{
  return (PcctPriv*)rte_mempool_get_priv(Pcct_ToMempool(pcct));
}

/** \brief Create a PIT-CS index.
 *  \param id identifier for debugging, up to 24 chars, must be unique.
 *  \param maxEntries maximum number of entries, should be (2^q-1).
 *  \param numaSocket where to allocate memory.
 *
 *  Caller must invoke \p Pit_Init and \p Cs_Init to initialize each table.
 */
Pcct*
Pcct_New(const char* id, uint32_t maxEntries, unsigned numaSocket);

/** \brief Release all memory.
 */
void
Pcct_Close(Pcct* pcct);

/** \brief Insert or find an entry.
 *  \param[out] isNew whether the entry is new
 */
PccEntry*
Pcct_Insert(Pcct* pcct, PccSearch* search, bool* isNew);

/** \brief Erase an entry.
 *  \sa PcctEraseBatch
 */
void
Pcct_Erase(Pcct* pcct, PccEntry* entry);

uint64_t
Pcct_AddToken_(Pcct* pcct, PccEntry* entry);

/** \brief Assign a token to an entry.
 *  \retval 0 No token available.
 *  \return New or existing token.
 */
static inline uint64_t
Pcct_AddToken(Pcct* pcct, PccEntry* entry)
{
  if (entry->hasToken) {
    return entry->token;
  }
  return Pcct_AddToken_(pcct, entry);
}

void
Pcct_RemoveToken_(Pcct* pcct, PccEntry* entry);

/** \brief Clear the token on an entry.
 */
static inline void
Pcct_RemoveToken(Pcct* pcct, PccEntry* entry)
{
  if (!entry->hasToken) {
    return;
  }
  Pcct_RemoveToken_(pcct, entry);
}

/** \brief Find an entry by token.
 *  \param token the token, only lower 48 bits are significant.
 */
PccEntry*
Pcct_FindByToken(const Pcct* pcct, uint64_t token);

// Burst size of PCCT erasing.
#define PCCT_ERASE_BURST 32

/** \brief Context for erasing several PCC entries.
 */
typedef struct PcctEraseBatch
{
  Pcct* pcct;
  int nEntries;
  void* objs[PCCT_ERASE_BURST * (2 + PCC_KEY_MAX_EXTS)];
} PcctEraseBatch;

/** \brief Create a PcctEraseBatch.
 *  \code
 *  PcctEraseBatch peb = PcctEraseBatch_New(pcct);
 *  PcctEraseBatch_Append(&peb, entry);
 *  PcctEraseBatch_Finish(&peb);
 *  \endcode
 */
#define PcctEraseBatch_New(thePcct)                                            \
  {                                                                            \
    .pcct = thePcct                                                            \
  }

void
PcctEraseBatch_EraseBurst_(PcctEraseBatch* peb);

/** \brief Add an entry for erasing.
 */
static inline void
PcctEraseBatch_Append(PcctEraseBatch* peb, PccEntry* entry)
{
  peb->objs[peb->nEntries] = entry;
  if (unlikely(++peb->nEntries == PCCT_ERASE_BURST)) {
    PcctEraseBatch_EraseBurst_(peb);
  }
}

/** \brief Erase entries.
 */
static inline void
PcctEraseBatch_Finish(PcctEraseBatch* peb)
{
  if (likely(peb->nEntries > 0)) {
    PcctEraseBatch_EraseBurst_(peb);
  }
  peb->pcct = NULL;
}

#endif // NDN_DPDK_CONTAINER_PCCT_PCCT_H
