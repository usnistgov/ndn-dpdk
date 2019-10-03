#ifndef NDN_DPDK_CONTAINER_TSHT_NODE_H
#define NDN_DPDK_CONTAINER_TSHT_NODE_H

/// \file

#include "../../core/common.h"
#include "../../core/urcu/urcu.h"

#include <rte_mempool.h>
#include <urcu/rculfhash.h>

typedef struct Tsht Tsht;

/** \brief A node in TSHT.
 *
 *  This must be embedded in an entry at offset 0 and named 'tshtNode'.
 *  Example:
 *  \code
 *  typedef MyEntry {
 *    TshtNode tshtNode;
 *    char key[32];
 *  } MyEntry;
 *  \endcode
 */
typedef struct TshtNode
{
  struct cds_lfht_node lfhtnode;
  struct rcu_head rcuhead;
} TshtNode;
static_assert(sizeof(TshtNode) == 32, "");

/** \brief A callback to finalize an erased node.
 */
typedef void (*Tsht_Finalize)(TshtNode* node, Tsht* ht);

/** \brief Argument to \c Tsht_Match.
 */
typedef struct cds_lfht_node* TshtMatchNodeRef;

/** \brief Recover TshtNode pointer from \c TshtMatchNodeRef.
 */
#define TshtMatch_GetNode(nodeRef)                                             \
  container_of(nodeRef, const TshtNode, lfhtnode)

/** \brief A callback to match node with key.
 *
 *  Example:
 *  \code
 *  bool ExampleMatch(TshtMatchNodeRef nodeRef, const void* key)
 *  {
 *    const MyEntry* entry = (const MyEntry*)TshtMatch_GetNode(node);
 *    return memcmp(entry->key, key, sizeof(entry->key)) == 0;
 *  }
 *  \endcode
 */
typedef bool (*Tsht_Match)(TshtMatchNodeRef nodeRef, const void* key);

#endif // NDN_DPDK_CONTAINER_TSHT_NODE_H
