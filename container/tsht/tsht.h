#ifndef NDN_DPDK_CONTAINER_TSHT_TSHT_H
#define NDN_DPDK_CONTAINER_TSHT_TSHT_H

/// \file

#include "node.h"

/** \brief A thread-safe hashtable.
 *
 *  Tsht* is struct rte_mempool* with \p TshtPriv is attached to its private data area.
 */
typedef struct Tsht
{
} Tsht;

/** \brief Cast Tsht* as rte_mempool*.
 */
static struct rte_mempool*
Tsht_ToMempool(const Tsht* ht)
{
  return (struct rte_mempool*)ht;
}

/** \brief rte_mempool private data for TSHT.
 */
typedef struct TshtPriv
{
  struct cds_lfht* lfht;    ///< URCU hashtable
  cds_lfht_match_fct match; ///< match function
  char head[0];             ///< private area for enclosing data structure
} TshtPriv;

/** \brief Access TshtPriv* struct.
 */
static TshtPriv*
Tsht_GetPriv(const Tsht* ht)
{
  return (TshtPriv*)rte_mempool_get_priv(Tsht_ToMempool(ht));
}

/** \brief Get private area after TshtPriv, and cast as T* type.
 */
#define Tsht_GetHead(ht, T) ((T*)Tsht_GetPriv((ht))->head)

/** \brief Create a TSHT.
 *  \param id identifier for debugging, must be unique.
 *  \param maxEntries maximum number of entries, should be (2^q-1).
 *  \param nBuckets number of buckets, must be (2^q).
 *  \param sizeofEntry size of the enclosed entry in each node.
 *  \param sizeofHead size of private area after TshtPriv.
 *  \param numaSocket where to allocate memory.
 */
Tsht* Tsht_New(const char* id, uint32_t maxEntries, uint32_t nBuckets,
               Tsht_Match match, size_t sizeofEntry, size_t sizeofHead,
               unsigned numaSocket);

/** \brief Release all memory.
 *  \pre Calling thread is registered as RCU read-side thread, but does not hold rcu_read_lock.
 *  \post \p ht pointer is no longer valid.
 *  \warning This function is not thread-safe.
 */
void Tsht_Close(Tsht* ht);

/** \brief Allocate an entry from TSHT's mempool.
 */
TshtEntryPtr Tsht_Alloc(Tsht* ht);

/** \brief Allocate an entry from TSHT's mempool, and cast as T* type.
 */
#define Tsht_AllocT(ht, T) ((T*)Tsht_Alloc((ht)))

/** \brief Deallocate an unused entry.
 */
void Tsht_Free(Tsht* ht, TshtEntryPtr entry);

/** \brief Insert an entry, or replace an entry with same key.
 *  \pre Calling thread holds rcu_read_lock.
 *  \retval true new entry inserted
 *  \retval false new entry replaced an old entry
 */
bool Tsht_Insert(Tsht* ht, uint64_t hash, const void* key,
                 TshtEntryPtr newEntry);

/** \brief Erase an entry.
 *  \pre Calling thread holds rcu_read_lock.
 *  \return whether success
 */
bool Tsht_Erase(Tsht* ht, TshtEntryPtr entry);

/** \brief Find an entry with specified key.
 *  \pre Calling thread holds rcu_read_lock.
 */
TshtEntryPtr Tsht_Find(Tsht* ht, uint64_t hash, const void* key);

/** \brief Find an entry with specified key, and cast as const T* type.
 */
#define Tsht_FindT(ht, hash, key, T) ((T*)Tsht_Find((ht), (hash), (key)))

#endif // NDN_DPDK_CONTAINER_TSHT_TSHT_H
