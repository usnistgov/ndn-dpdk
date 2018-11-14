#ifndef NDN_DPDK_IFACE_ETHFACE_ETH_FACE_H
#define NDN_DPDK_IFACE_ETHFACE_ETH_FACE_H

/// \file

#include "../../dpdk/ethdev.h"
#include "../face.h"
#include <rte_ether.h>

/** \brief Ethernet face private data.
 */
typedef struct EthFacePriv
{
  struct ether_hdr txHdr;
  uint16_t port;
} EthFacePriv;

uint16_t EthFace_TxBurst(Face* face, struct rte_mbuf** pkts, uint16_t nPkts);

#endif // NDN_DPDK_IFACE_ETHFACE_ETH_FACE_H
