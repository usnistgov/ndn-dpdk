#ifndef NDN_DPDK_CONTAINER_FIB_FIB_H
#define NDN_DPDK_CONTAINER_FIB_FIB_H

/// \file

#include "../tsht/tsht.h"
#include "entry.h"

/** \brief The Forwarding Information Base (FIB).
 */
typedef Tsht Fib;

/** \brief TSHT private data for FIB.
 */
typedef struct FibPriv
{
  int startDepth; ///< starting depth ('M' in 2-stage LPM paper)
} FibPriv;

#define Fib_GetPriv(fib) Tsht_GetHead(fib, FibPriv)

/** \brief Create a FIB.
 *  \param id identifier for debugging, must be unique.
 *  \param maxEntries maximum number of entries, should be (2^q-1).
 *  \param nBuckets number of hashtable buckets, must be (2^q).
 *  \param numaSocket where to allocate memory.
 */
Fib* Fib_New(const char* id, uint32_t maxEntries, uint32_t nBuckets,
             unsigned numaSocket, uint8_t startDepth);

/** \brief Release all memory.
 */
static inline void
Fib_Close(Fib* fib)
{
  Tsht_Close(fib);
}

/** \brief Allocate and zero a FIB entry from mempool.
 */
FibEntry* Fib_Alloc(Fib* fib);

/** \brief Deallocate an unused FIB entry.
 */
static inline void
Fib_Free(Fib* fib, FibEntry* entry)
{
  Tsht_Free(fib, entry);
}

/** \brief Insert a FIB entry, or replace an existing entry with same name.
 *  \param entry an entry allocated from \p Fib_Alloc.
 *  \retval true new entry inserted.
 *  \retval false old entry replaced by new entry.
 *  \pre Calling thread holds rcu_read_lock.
 */
bool Fib_Insert(Fib* fib, FibEntry* entry);

/** \brief Erase a FIB entry of given name.
 *  \return whether success
 *  \pre Calling thread holds rcu_read_lock.
 */
static inline void
Fib_Erase(Fib* fib, FibEntry* entry)
{
  Tsht_Erase(fib, entry);
}

/** \brief Perform exact match.
 *  \pre Calling thread holds rcu_read_lock, which must be retained until it stops
 *       using the returned entry.
 */
const FibEntry* Fib_Find(Fib* fib, uint16_t nameL, const uint8_t* nameV);

/** \brief Perform longest prefix match.
 *  \pre Calling thread holds rcu_read_lock, which must be retained until it stops
 *       using the returned entry.
 */
const FibEntry* Fib_Lpm(Fib* fib, const Name* name);

#endif // NDN_DPDK_CONTAINER_FIB_FIB_H
