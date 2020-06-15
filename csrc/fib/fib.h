#ifndef NDN_DPDK_FIB_FIB_H
#define NDN_DPDK_FIB_FIB_H

/// \file

#include "entry.h"

/** \brief A partition of the Forwarding Information Base (FIB).
 *
 *  Fib* is struct rte_mempool* with \c FibPriv is attached to its private data area.
 */
typedef struct Fib
{
} Fib;

/** \brief Cast Fib* as rte_mempool*.
 */
static inline struct rte_mempool*
Fib_ToMempool(const Fib* fib)
{
  return (struct rte_mempool*)fib;
}

/** \brief TSHT private data for FIB.
 */
typedef struct FibPriv
{
  struct cds_lfht* lfht; ///< URCU hashtable
  int startDepth;        ///< starting depth ('M' of 2-stage LPM algorithm)
  uint32_t insertSeqNum;
} FibPriv;

/** \brief Access FibPriv* struct.
 */
static inline FibPriv*
Fib_GetPriv(const Fib* fib)
{
  return (FibPriv*)rte_mempool_get_priv(Fib_ToMempool(fib));
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
 *  \pre Calling thread is registered as RCU read-side thread, but does not hold rcu_read_lock.
 *  \post \p ht pointer is no longer valid.
 *  \warning This function is non-thread-safe.
 */
void
Fib_Close(Fib* fib);

/** \brief Allocate FIB entries from mempool.
 */
bool
Fib_AllocBulk(Fib* fib, FibEntry* entries[], unsigned count);

/** \brief Deallocate an unused FIB entry.
 */
void
Fib_Free(Fib* fib, FibEntry* entry);

typedef enum Fib_FreeOld
{
  Fib_FreeOld_MustNotExist = -1,
  Fib_FreeOld_No = 0,
  Fib_FreeOld_Yes = 1,
  Fib_FreeOld_YesIfExists = 2,
} Fib_FreeOld;

/** \brief Insert a FIB entry, or replace an existing entry with same name.
 *  \param entry an entry allocated from \c Fib_Alloc.
 *  \pre Calling thread holds rcu_read_lock.
 */
void
Fib_Insert(Fib* fib,
           FibEntry* entry,
           Fib_FreeOld freeVirt,
           Fib_FreeOld freeReal);

/** \brief Erase given FIB entry.
 *  \pre Calling thread holds rcu_read_lock.
 */
void
Fib_Erase(Fib* fib,
          FibEntry* entry,
          Fib_FreeOld freeVirt,
          Fib_FreeOld freeReal);

/** \brief Retrieve FIB entry.
 *  \pre Calling thread holds rcu_read_lock, which must be retained until it stops
 *       using the returned entry.
 *  \return Virtual or real entry, or NULL if it does not exist.
 */
FibEntry*
Fib_Get(Fib* fib, LName name, uint64_t hash);

static inline FibEntry*
Fib_Get_(Fib* fib, uint16_t nameL, const uint8_t* nameV, uint64_t hash)
{
  LName name = { .length = nameL, .value = nameV };
  return Fib_Get(fib, name, hash);
}

/** \brief Perform exact match.
 *  \pre Calling thread holds rcu_read_lock, which must be retained until it stops
 *       using the returned entry.
 *  \return Real entry, or NULL if it does not exist.
 */
static inline FibEntry*
Fib_Find(Fib* fib, LName name, uint64_t hash)
{
  return FibEntry_GetReal(Fib_Get(fib, name, hash));
}

static inline FibEntry*
Fib_Find_(Fib* fib, uint16_t nameL, const uint8_t* nameV, uint64_t hash)
{
  LName name = { .length = nameL, .value = nameV };
  return Fib_Find(fib, name, hash);
}

FibEntry*
Fib_Lpm_(Fib* fib, const PName* name, const uint8_t* nameV);

/** \brief Perform longest prefix match.
 *  \pre Calling thread holds rcu_read_lock, which must be retained until it stops
 *       using the returned entry.
 *  \return Real entry, or NULL if it does not exist.
 */
static inline FibEntry*
Fib_Lpm(Fib* fib, const Name* name)
{
  return Fib_Lpm_(fib, &name->p, name->v);
}

#endif // NDN_DPDK_FIB_FIB_H
