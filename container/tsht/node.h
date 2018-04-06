#ifndef NDN_DPDK_CONTAINER_TSHT_NODE_H
#define NDN_DPDK_CONTAINER_TSHT_NODE_H

/// \file

#include "../../core/common.h"
#include "../../core/urcu/urcu.h"

#include <rte_mempool.h>
#include <urcu/rculfhash.h>

typedef struct Tsht Tsht;

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

/** \brief A callback to finalize an erased entry.
 */
typedef void (*Tsht_Finalize)(TshtEntryPtr entry, Tsht* ht);

/** \brief Get the node enclosing an entry.
 *  \param entry1 a TshtEntryPtr.
 */
#define TshtNode_FromEntry(entry1)                                             \
  ((TshtNode*)RTE_PTR_SUB((entry1), offsetof(TshtNode, entry)))

/** \brief Argument to \c Tsht_Match.
 */
typedef struct cds_lfht_node* TshtMatchNode;

/** \brief Recover entry from \c TshtMatchNode.
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

#endif // NDN_DPDK_CONTAINER_TSHT_NODE_H
