#ifndef NDN_DPDK_CONTAINER_TSHT_TSHT_H
#define NDN_DPDK_CONTAINER_TSHT_TSHT_H

/// \file

#include "node.h"

/** \brief A thread-safe hashtable.
 *
 *  Tsht* is struct rte_mempool* with \c TshtPriv is attached to its private data area.
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
  Tsht_Finalize finalize;   ///< finalize function
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
 *  \param sizeofEntry size of the entry enclosing TshtNode.
 *  \param sizeofHead size of private area after TshtPriv.
 *  \param numaSocket where to allocate memory.
 */
Tsht*
Tsht_New(const char* id,
         uint32_t maxEntries,
         uint32_t nBuckets,
         Tsht_Match match,
         Tsht_Finalize finalize,
         size_t sizeofEntry,
         size_t sizeofHead,
         unsigned numaSocket);

/** \brief Release all memory.
 *  \pre Calling thread is registered as RCU read-side thread, but does not hold rcu_read_lock.
 *  \post \p ht pointer is no longer valid.
 *  \warning This function is non-thread-safe.
 */
void
Tsht_Close(Tsht* ht);

/** \brief Allocate nodes from TSHT's mempool.
 */
bool
Tsht_AllocBulk(Tsht* ht, TshtNode* nodes[], unsigned count);

/** \brief Deallocate an unused node.
 *
 *  \c Tsht_Finalize will not be invoked for this node.
 */
void
Tsht_Free(Tsht* ht, TshtNode* node);

/** \brief Insert a node, or replace a node with same key.
 *  \pre Calling thread holds rcu_read_lock.
 *  \retval true new node inserted
 *  \retval false new node replaced an old node
 *
 *  \c Tsht_Finalize will be invoked for the old node when it can be released.
 */
bool
Tsht_Insert(Tsht* ht, uint64_t hash, const void* key, TshtNode* newNode);

/** \brief Erase an node.
 *  \pre Calling thread holds rcu_read_lock.
 *  \return whether success
 *
 *  \c Tsht_Finalize will be invoked for this node when it can be released.
 */
bool
Tsht_Erase(Tsht* ht, TshtNode* node);

/** \brief Find an entry with specified key.
 *  \pre Calling thread holds rcu_read_lock.
 */
TshtNode*
Tsht_Find(Tsht* ht, uint64_t hash, const void* key);

/** \brief Find an entry with specified key, and cast as const T* type.
 */
#define Tsht_FindT(ht, hash, key, T)                                           \
  __extension__({                                                              \
    static_assert(offsetof(T, tshtNode) == 0, "");                             \
    ((T*)Tsht_Find((ht), (hash), (key)));                                      \
  })

#endif // NDN_DPDK_CONTAINER_TSHT_TSHT_H
