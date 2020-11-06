#ifndef NDNDPDK_ETHFACE_FACE_H
#define NDNDPDK_ETHFACE_FACE_H

/** @file */

#include "../dpdk/ethdev.h"
#include "../iface/face.h"
#include "../iface/rxloop.h"
#include "locator.h"

/**
 * @brief Ethernet face private data.
 *
 * This struct doubles as RxGroup when not using EthRxTable.
 */
typedef struct EthFacePriv
{
  RxGroup flowRxg;
  uint16_t port;
  uint16_t rxQueue;
  FaceID faceID;
  EthTxHdr txHdr;

  struct cds_hlist_node rxtNode;
  EthRxMatch rxMatch;
} EthFacePriv;

__attribute__((nonnull)) uint16_t
EthFace_TxBurst(Face* face, struct rte_mbuf** pkts, uint16_t nPkts);

/** @brief Setup rte_flow on EthDev for hardware dispatching. */
__attribute__((nonnull)) struct rte_flow*
EthFace_SetupFlow(EthFacePriv* priv, const EthLocator* loc, struct rte_flow_error* error);

__attribute__((nonnull)) uint16_t
EthFace_FlowRxBurst(RxGroup* flowRxg, struct rte_mbuf** pkts, uint16_t nPkts);

#endif // NDNDPDK_ETHFACE_FACE_H
