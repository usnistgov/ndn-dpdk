#ifndef NDNDPDK_ETHFACE_FACE_H
#define NDNDPDK_ETHFACE_FACE_H

/** @file */

#include "../dpdk/ethdev.h"
#include "../iface/face.h"
#include "../iface/rxloop.h"
#include "locator.h"

/** @brief rte_flow hardware assisted RX dispatching. */
typedef struct EthRxFlow
{
  RxGroup base;
  FaceID faceID;
  uint16_t port;
  uint16_t queue;
  uint16_t hdrLen;
} __rte_cache_aligned EthRxFlow;

/** @brief Ethernet face private data. */
typedef struct EthFacePriv
{
  EthRxFlow rxf[RXPROC_MAX_THREADS];
  EthTxHdr txHdr;
  FaceID faceID;
  uint16_t port;

  struct cds_hlist_node rxtNode;
  EthRxMatch rxMatch;
} EthFacePriv;

/** @brief Setup rte_flow on EthDev for hardware dispatching. */
__attribute__((nonnull)) struct rte_flow*
EthFace_SetupFlow(EthFacePriv* priv, int index, uint16_t queue, const EthLocator* loc,
                  struct rte_flow_error* error);

__attribute__((nonnull)) uint16_t
EthFace_TxBurst(Face* face, struct rte_mbuf** pkts, uint16_t nPkts);

#endif // NDNDPDK_ETHFACE_FACE_H
