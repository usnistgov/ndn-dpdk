#ifndef NDN_DPDK_CONTAINER_FIB_FIB_H
#define NDN_DPDK_CONTAINER_FIB_FIB_H

/// \file

#include "../tsht/tsht.h"
#include "entry.h"

/** \brief A partition of the Forwarding Information Base (FIB).
 *
 *  Fib* is Tsht* with \c FibPriv at its 'head' field.
 */
typedef struct Fib
{
} Fib;

/** \brief Cast Fib* as Tsht*.
 */
static Tsht*
Fib_ToTsht(const Fib* fib)
{
  return (Tsht*)fib;
}

/** \brief TSHT private data for FIB.
 */
typedef struct FibPriv
{
  int startDepth; ///< starting depth ('M' in 2-stage LPM paper)
} FibPriv;

/** \brief Access FibPriv* struct.
 */
static FibPriv*
Fib_GetPriv(const Fib* fib)
{
  return Tsht_GetHead(Fib_ToTsht(fib), FibPriv);
}

/** \brief Create a FIB.
 *  \param id identifier for debugging, must be unique.
 *  \param maxEntries maximum number of entries, should be (2^q-1).
 *  \param nBuckets number of hashtable buckets, must be (2^q).
 *  \param numaSocket where to allocate memory.
 */
Fib*
Fib_New(const char* id,
        uint32_t maxEntries,
        uint32_t nBuckets,
        unsigned numaSocket,
        uint8_t startDepth);

/** \brief Release all memory.
 */
static void
Fib_Close(Fib* fib)
{
  Tsht_Close(Fib_ToTsht(fib));
}

/** \brief Allocate FIB entries from mempool.
 */
static bool
Fib_AllocBulk(Fib* fib, FibEntry* entries[], unsigned count)
{
  return Tsht_AllocBulk(Fib_ToTsht(fib), (TshtNode**)entries, count);
}

/** \brief Deallocate an unused FIB entry.
 */
static void
Fib_Free(Fib* fib, FibEntry* entry)
{
  assert(entry->strategy == NULL);
  Tsht_Free(Fib_ToTsht(fib), &entry->tshtNode);
}

/** \brief Insert a FIB entry, or replace an existing entry with same name.
 *  \param entry an entry allocated from \c Fib_Alloc.
 *  \retval true new entry inserted.
 *  \retval false old entry replaced by new entry.
 *  \pre Calling thread holds rcu_read_lock.
 */
bool
Fib_Insert(Fib* fib, FibEntry* entry);

/** \brief Erase given FIB entry.
 *  \return whether success
 *  \pre Calling thread holds rcu_read_lock.
 */
static void
Fib_Erase(Fib* fib, FibEntry* entry)
{
  Tsht_Erase(Fib_ToTsht(fib), &entry->tshtNode);
}

/** \brief Perform exact match.
 *  \pre Calling thread holds rcu_read_lock, which must be retained until it stops
 *       using the returned entry.
 */
static const FibEntry*
Fib_Find(Fib* fib, LName name, uint64_t hash)
{
  return Tsht_FindT(Fib_ToTsht(fib), hash, &name, FibEntry);
}

static const FibEntry*
Fib_Find_(Fib* fib, uint16_t nameL, const uint8_t* nameV, uint64_t hash)
{
  LName name = { .length = nameL, .value = nameV };
  return Fib_Find(fib, name, hash);
}

const FibEntry*
Fib_Lpm_(Fib* fib, const PName* name, const uint8_t* nameV);

/** \brief Perform longest prefix match.
 *  \pre Calling thread holds rcu_read_lock, which must be retained until it stops
 *       using the returned entry.
 */
static const FibEntry*
Fib_Lpm(Fib* fib, const Name* name)
{
  return Fib_Lpm_(fib, &name->p, name->v);
}

#endif // NDN_DPDK_CONTAINER_FIB_FIB_H
