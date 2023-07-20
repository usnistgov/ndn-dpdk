#ifndef NDNDPDK_ETHFACE_FACE_H
#define NDNDPDK_ETHFACE_FACE_H

/** @file */

#include "../iface/rxloop.h"
#include "locator.h"

/** @brief rte_flow hardware assisted RX dispatching. */
typedef struct EthRxFlow {
  RxGroup base;
  FaceID faceID;
  uint16_t port;
  uint16_t queue;
  uint16_t hdrLen;
  EthRxMatch* rxMatch; // when not flow isolated
} __rte_cache_aligned EthRxFlow;

/** @brief Ethernet face private data. */
typedef struct EthFacePriv {
  EthRxFlow rxf[MaxFaceRxThreads];
  EthTxHdr txHdr;
  FaceID faceID;
  uint16_t port;

  struct cds_hlist_node rxtNode;
  EthRxMatch rxMatch;
} EthFacePriv;

/** @brief Setup rte_flow on EthDev for hardware dispatching. */
__attribute__((nonnull)) struct rte_flow*
EthFace_SetupFlow(EthFacePriv* priv, const uint16_t queues[], int nQueues, const EthLocator* loc,
                  bool isolated, struct rte_flow_error* error);

/** @brief Setup RX for memif. */
__attribute__((nonnull)) void
EthFace_SetupRxMemif(EthFacePriv* priv, const EthLocator* loc);

__attribute__((nonnull)) uint16_t
EthFace_TxBurst(Face* face, struct rte_mbuf** pkts, uint16_t nPkts);

#endif // NDNDPDK_ETHFACE_FACE_H
