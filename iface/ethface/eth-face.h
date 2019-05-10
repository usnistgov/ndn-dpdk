#ifndef NDN_DPDK_IFACE_ETHFACE_ETH_FACE_H
#define NDN_DPDK_IFACE_ETHFACE_ETH_FACE_H

/// \file

#include "../../dpdk/ethdev.h"
#include "../face.h"
#include "../rxloop.h"
#include <rte_ether.h>
#include <rte_flow.h>

/** \brief Ethernet face private data.
 *
 *  This struct doubles as RxGroup when not using EthRxTable.
 */
typedef struct EthFacePriv
{
  RxGroup flowRxg;
  struct ether_hdr txHdr;
  uint16_t port;
  uint16_t rxQueue;
  FaceId faceId;
} EthFacePriv;

uint16_t
EthFace_TxBurst(Face* face, struct rte_mbuf** pkts, uint16_t nPkts);

/** \brief Setup rte_flow on EthDev for hardware dispatching.
 */
struct rte_flow*
EthFace_SetupFlow(EthFacePriv* priv, struct rte_flow_error* error);

uint16_t
EthFace_FlowRxBurst(RxGroup* flowRxg, struct rte_mbuf** pkts, uint16_t nPkts);

#endif // NDN_DPDK_IFACE_ETHFACE_ETH_FACE_H
