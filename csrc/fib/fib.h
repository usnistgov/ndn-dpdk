#ifndef NDNDPDK_FIB_FIB_H
#define NDNDPDK_FIB_FIB_H

/** @file */

#include "entry.h"

/** @brief A partition of the Forwarding Information Base (FIB). */
typedef struct Fib
{
  struct rte_mempool* mp; ///< entry mempool
  struct cds_lfht* lfht;  ///< URCU hashtable
  int startDepth;         ///< starting depth ('M' of 2-stage LPM algorithm)
  uint32_t insertSeqNum;
} Fib;

/** @brief Allocate FIB entries from mempool. */
__attribute__((nonnull)) bool
Fib_AllocBulk(Fib* fib, FibEntry* entries[], unsigned count);

/** @brief Deallocate an unused FIB entry. */
__attribute__((nonnull)) void
Fib_Free(Fib* fib, FibEntry* entry);

/**
 * @brief Delete all entries.
 * @pre Calling thread is registered as RCU read-side thread, but does not hold rcu_read_lock.
 */
__attribute__((nonnull)) void
Fib_Clear(Fib* fib);

typedef enum Fib_FreeOld
{
  Fib_FreeOld_MustNotExist = -1,
  Fib_FreeOld_No = 0,
  Fib_FreeOld_Yes = 1,
  Fib_FreeOld_YesIfExists = 2,
} Fib_FreeOld;

/**
 * @brief Insert a FIB entry, or replace an existing entry with same name.
 * @param entry an entry allocated from @c Fib_Alloc.
 * @pre Calling thread holds rcu_read_lock.
 */
__attribute__((nonnull)) void
Fib_Insert(Fib* fib, FibEntry* entry, Fib_FreeOld freeVirt, Fib_FreeOld freeReal);

/**
 * @brief Erase given FIB entry.
 * @pre Calling thread holds rcu_read_lock.
 */
__attribute__((nonnull)) void
Fib_Erase(Fib* fib, FibEntry* entry, Fib_FreeOld freeVirt, Fib_FreeOld freeReal);

/**
 * @brief Retrieve FIB entry.
 * @pre Calling thread holds rcu_read_lock, which must be retained until it stops
 *      using the returned entry.
 * @return Virtual or real entry, or NULL if it does not exist.
 */
__attribute__((nonnull)) FibEntry*
Fib_Get(Fib* fib, LName name, uint64_t hash);

/**
 * @brief Perform exact match.
 * @pre Calling thread holds rcu_read_lock, which must be retained until it stops
 *      using the returned entry.
 * @return Real entry, or NULL if it does not exist.
 */
__attribute__((nonnull)) static __rte_always_inline FibEntry*
Fib_Find(Fib* fib, LName name, uint64_t hash)
{
  return FibEntry_GetReal(Fib_Get(fib, name, hash));
}

/**
 * @brief Perform longest prefix match.
 * @pre Calling thread holds rcu_read_lock, which must be retained until it stops
 *      using the returned entry.
 * @return Real entry, or NULL if it does not exist.
 */
__attribute__((nonnull)) FibEntry*
Fib_Lpm(Fib* fib, const PName* name);

#endif // NDNDPDK_FIB_FIB_H
