#ifndef NDN_DPDK_CONTAINER_TSHT_TSHT_H
#define NDN_DPDK_CONTAINER_TSHT_TSHT_H

/// \file

#include "../../core/common.h"
#include "../../core/urcu/urcu.h"

#include <urcu/rculfhash.h>

/** \brief A node in TSHT.
 */
typedef struct TshtNode
{
  struct cds_lfht_node lfhtnode;
  struct rcu_head rcuhead;
  char entry[0];
} TshtNode;

/** \brief Represents an entry enclosed in TshtNode.
 */
typedef void* TshtEntryPtr;

/** \brief Get the node enclosing an entry.
 *  \param entry1 a TshtEntryPtr.
 */
#define TshtNode_FromEntry(entry1)                                             \
  ((TshtNode*)((char*)(entry1)-offsetof(TshtNode, entry)))

/** \brief Argument to \p Tsht_Match.
 */
typedef struct cds_lfht_node* TshtMatchNode;

/** \brief Recover entry from \p TshtMatchNode.
 *  \tparam T type of entry.
 */
#define TshtMatch_GetEntry(node, T)                                            \
  ((const T*)caa_container_of((node), TshtNode, lfhtnode)->entry)

/** \brief A callback to match entry with key.
 *
 *  Example:
 *  \code
 *  bool ExampleMatch(TshtMatchNode node, const void* key)
 *  {
 *    const MyEntry* entry = TshtMatch_GetEntry(node, MyEntry);
 *    return memcmp(entry, key, sizeof(*entry)) == 0;
 *  }
 *  \endcode
 */
typedef bool (*Tsht_Match)(TshtMatchNode node, const void* key);

/** \brief A thread-safe hashtable.
 *
 *  \p TshtPriv is attached to the private data area of this mempool.
 */
typedef struct rte_mempool Tsht;

/** \brief rte_mempool private data for TSHT.
 */
typedef struct TshtPriv
{
  struct cds_lfht* lfht;    ///< URCU hashtable
  cds_lfht_match_fct match; ///< match function
  char head[0];             ///< private area for enclosing data structure
} TshtPriv;

#define Tsht_GetPriv(ht) ((TshtPriv*)rte_mempool_get_priv((ht)))

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
