#ifndef NDN_DPDK_ETHFACE_ETH_FACE_H
#define NDN_DPDK_ETHFACE_ETH_FACE_H

/// \file

#include "../dpdk/ethdev.h"
#include "../iface/face.h"
#include "../iface/rxloop.h"
#include <rte_flow.h>

#define NDN_ETHERTYPE 0x8624

typedef struct EthFaceEtherHdr
{
  struct rte_ether_hdr eth;
  struct rte_vlan_hdr vlan0;
  struct rte_vlan_hdr vlan1;
} __rte_packed __rte_aligned(2) EthFaceEtherHdr;

uint8_t
EthFaceEtherHdr_Init(EthFaceEtherHdr* hdr,
                     const struct rte_ether_addr* local,
                     const struct rte_ether_addr* remote,
                     uint16_t vlan0,
                     uint16_t vlan1);

/** \brief Ethernet face private data.
 *
 *  This struct doubles as RxGroup when not using EthRxTable.
 */
typedef struct EthFacePriv
{
  RxGroup flowRxg;
  EthFaceEtherHdr txHdr;
  uint16_t port;
  uint16_t rxQueue;
  FaceId faceId;
  uint8_t txHdrLen;
} EthFacePriv;

uint16_t
EthFace_TxBurst(Face* face, struct rte_mbuf** pkts, uint16_t nPkts);

/** \brief Setup rte_flow on EthDev for hardware dispatching.
 */
struct rte_flow*
EthFace_SetupFlow(EthFacePriv* priv, struct rte_flow_error* error);

uint16_t
EthFace_FlowRxBurst(RxGroup* flowRxg, struct rte_mbuf** pkts, uint16_t nPkts);

#endif // NDN_DPDK_ETHFACE_ETH_FACE_H
