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
} FibPriv;

#define Fib_GetPriv(fib) Tsht_GetHead(fib, FibPriv)

/** \brief Create a FIB.
 *  \param id identifier for debugging, must be unique.
 *  \param maxEntries maximum number of entries, should be (2^q-1).
 *  \param nBuckets number of hashtable buckets, must be (2^q).
 *  \param numaSocket where to allocate memory.
 */
Fib* Fib_New(const char* id, uint32_t maxEntries, uint32_t nBuckets,
             unsigned numaSocket);

/** \brief Release all memory.
 */
void Fib_Close(Fib* fib);

typedef enum FibInsertResult {
  FIB_INSERT_REPLACE = 0,     ///< old entry replaced by new entry
  FIB_INSERT_NEW = 1,         ///< new entry inserted
  FIB_INSERT_ALLOC_ERROR = 2, ///< allocation error
} FibInsertResult;

/** \brief Insert a FIB entry, or replace an existing entry with same name.
 *  \param entry the entry, will be copied.
 *  \pre Calling thread is registered as RCU read-side thread.
 */
FibInsertResult Fib_Insert(Fib* fib, const FibEntry* entry);

/** \brief Erase a FIB entry of given name.
 *  \return whether success
 *  \pre Calling thread is registered as RCU read-side thread.
 */
bool Fib_Erase(Fib* fib, uint16_t nameL, const uint8_t* nameV);

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
