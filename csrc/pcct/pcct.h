#ifndef NDNDPDK_PCCT_PCCT_H
#define NDNDPDK_PCCT_PCCT_H

/** @file */

#include "cs-struct.h"
#include "pcc-entry.h"
#include "pit-struct.h"

enum
{
  PccTokenSize = 6,
  PccTokenMask = ((uint64_t)1 << (PccTokenSize * 8)) - 1,
};

/** @brief The PIT-CS Composite Table (PCCT). */
typedef struct Pcct
{
  struct rte_mempool* mp;   ///< entry mempool
  PccEntry* keyHt;          ///< key hashtable
  struct rte_hash* tokenHt; ///< token hashtable
  uint64_t lastToken;       ///< last assigned token

  Pit pit;
  Cs cs;

  uint32_t nKeyHtBuckets;
} Pcct;

static __rte_always_inline Pcct*
Pcct_FromPit(const Pit* pit)
{
  return container_of(pit, Pcct, pit);
}

static __rte_always_inline Pcct*
Pcct_FromCs(const Cs* cs)
{
  return container_of(cs, Pcct, cs);
}

/**
 * @brief Initialize keyHt and tokenHt.
 * @param id memzone identifier, must be unique.
 * @param maxEntries PCCT capacity; hashtable capacity will be calculated accordingly.
 * @return whether success. Error code is in @c rte_errno .
 */
bool
Pcct_Init(Pcct* pcct, const char* id, uint32_t maxEntries, int numaSocket);

/**
 * @brief Clear keyHt and tokenHt, and free cached Data.
 * @post Pcct mempool can be deallocated.
 */
void
Pcct_Clear(Pcct* pcct);

/**
 * @brief Insert or find an entry.
 * @param[out] isNew whether the entry is new
 */
__attribute__((nonnull)) PccEntry*
Pcct_Insert(Pcct* pcct, const PccSearch* search, bool* isNew);

/**
 * @brief Erase an entry.
 * @sa PcctEraseBatch
 */
__attribute__((nonnull)) void
Pcct_Erase(Pcct* pcct, PccEntry* entry);

__attribute__((nonnull)) uint64_t
Pcct_AddToken_(Pcct* pcct, PccEntry* entry);

/**
 * @brief Assign a token to an entry.
 * @retval 0 No token available.
 * @return New or existing token.
 */
__attribute__((nonnull)) static __rte_always_inline uint64_t
Pcct_AddToken(Pcct* pcct, PccEntry* entry)
{
  if (entry->hasToken) {
    return entry->token;
  }
  return Pcct_AddToken_(pcct, entry);
}

__attribute__((nonnull)) void
Pcct_RemoveToken_(Pcct* pcct, PccEntry* entry);

/** @brief Clear the token on an entry. */
__attribute__((nonnull)) static __rte_always_inline void
Pcct_RemoveToken(Pcct* pcct, PccEntry* entry)
{
  if (!entry->hasToken) {
    return;
  }
  Pcct_RemoveToken_(pcct, entry);
}

/**
 * @brief Find an entry by token.
 * @param token the token, only lower 48 bits are significant.
 */
__attribute__((nonnull)) PccEntry*
Pcct_FindByToken(const Pcct* pcct, uint64_t token);

// Burst size of PCCT erasing.
#define PCCT_ERASE_BURST 32

/** @brief Context for erasing several PCC entries. */
typedef struct PcctEraseBatch
{
  Pcct* pcct;
  int nEntries;
  void* objs[PCCT_ERASE_BURST * (2 + PccKeyMaxExts)];
} PcctEraseBatch;

/**
 * @brief Create a PcctEraseBatch.
 * @code
 * PcctEraseBatch peb = PcctEraseBatch_New(pcct);
 * PcctEraseBatch_Append(&peb, entry);
 * PcctEraseBatch_Finish(&peb);
 * @endcode
 */
__attribute__((nonnull)) static inline PcctEraseBatch
PcctEraseBatch_New(Pcct* pcct)
{
  return (PcctEraseBatch){ .pcct = pcct };
}

__attribute__((nonnull)) void
PcctEraseBatch_EraseBurst_(PcctEraseBatch* peb);

/** @brief Add an entry for erasing. */
__attribute__((nonnull)) static inline void
PcctEraseBatch_Append(PcctEraseBatch* peb, PccEntry* entry)
{
  peb->objs[peb->nEntries] = entry;
  if (unlikely(++peb->nEntries == PCCT_ERASE_BURST)) {
    PcctEraseBatch_EraseBurst_(peb);
  }
}

/** @brief Erase entries. */
__attribute__((nonnull)) static inline void
PcctEraseBatch_Finish(PcctEraseBatch* peb)
{
  if (likely(peb->nEntries > 0)) {
    PcctEraseBatch_EraseBurst_(peb);
  }
  peb->pcct = NULL;
}

#endif // NDNDPDK_PCCT_PCCT_H
