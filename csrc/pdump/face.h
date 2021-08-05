#ifndef NDNDPDK_PDUMP_FACE_H
#define NDNDPDK_PDUMP_FACE_H

/** @file */

#include "../core/urcu.h"
#include "../iface/faceid.h"
#include "../vendor/pcg_basic.h"
#include "enum.h"
#include <urcu-pointer.h>

/** @brief Packet dump from a face RxProc or TxProc. */
typedef struct PdumpFace
{
  struct rte_mempool* directMp;
  struct rte_ring* queue;
  pcg32_random_t rng;
  rte_be16_t sllType;
  uint32_t sample[PdumpMaxNames];
  uint16_t nameL[PdumpMaxNames];
  uint8_t nameV[PdumpMaxNames * NameMaxLength];
} PdumpFace;

/**
 * @brief Submit packets for potential dumping.
 * @post If a packet is chosen for dumping, it is copied (with payload) to new mbuf.
 *       @p pkts are not freed or referenced.
 */
__attribute__((nonnull)) void
PdumpFace_Process(PdumpFace* pd, FaceID id, struct rte_mbuf** pkts, uint16_t count);

/** @brief A pointer to PdumpFace. */
typedef struct PdumpFaceRef
{
  PdumpFace* pd;
} PdumpFaceRef;

/** @brief Assign or clear PdumpFace in PdumpFaceRef. */
__attribute__((nonnull(1))) static inline PdumpFace*
PdumpFaceRef_Set(PdumpFaceRef* pdr, PdumpFace* pd)
{
  return rcu_xchg_pointer(&pdr->pd, pd);
}

/**
 * @brief Submit packets for potential dumping if dumper is enabled.
 * @pre Calling thread holds rcu_read_lock.
 */
__attribute__((nonnull)) static __rte_always_inline void
PdumpFaceRef_Process(PdumpFaceRef* pdr, FaceID id, struct rte_mbuf** pkts, uint16_t count)
{
  PdumpFace* pd = rcu_dereference(pdr->pd);
  if (pd == NULL) {
    return;
  }
  PdumpFace_Process(pd, id, pkts, count);
}

#endif // NDNDPDK_PDUMP_FACE_H
