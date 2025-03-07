#ifndef NDNDPDK_ETHFACE_FACE_H
#define NDNDPDK_ETHFACE_FACE_H

/** @file */

#include "passthru.h"
#include "rxmatch.h"
#include "txhdr.h"
#include <urcu/rculist.h>

/** @brief rte_flow hardware assisted RX dispatching. */
typedef struct EthRxFlow {
  RxGroup base;
  FaceID faceID;
  uint16_t port;
  uint16_t queue;
  uint8_t hdrLen;
} __rte_cache_aligned EthRxFlow;

/** @brief Ethernet face private data. */
typedef struct EthFacePriv {
  EthRxFlow rxf[MaxFaceRxThreads];
  EthPassthru passthru;
  EthTxHdr txHdr;
  FaceID faceID;
  uint16_t port;

  struct cds_list_head rxtNode;
  EthRxMatch rxMatch;
} EthFacePriv;

__attribute__((nonnull)) static __rte_always_inline FaceID
EthFace_RxMbufFaceID(struct rte_mbuf* m) {
  enum {
    MbufHasMark = RTE_MBUF_F_RX_FDIR | RTE_MBUF_F_RX_FDIR_ID,
  };
  if ((m->ol_flags & MbufHasMark) != MbufHasMark) {
    return 0;
  }
  FaceID id = m->hash.fdir.hi;
  __rte_assume(id != 0);
  return id;
}

/** @brief Setup rte_flow on EthDev for hardware dispatching. */
__attribute__((nonnull)) struct rte_flow*
EthFace_SetupFlow(EthFacePriv* priv, const uint16_t queues[], int nQueues, const EthLocator* loc,
                  bool isolated, EthFlowFlags flowFlags, struct rte_flow_error* error);

/** @brief Setup RX for memif. */
__attribute__((nonnull)) void
EthFace_SetupRxMemif(EthFacePriv* priv, const EthLocator* loc);

__attribute__((nonnull)) uint16_t
EthFace_TxBurst(Face* face, struct rte_mbuf** pkts, uint16_t nPkts);

#endif // NDNDPDK_ETHFACE_FACE_H
