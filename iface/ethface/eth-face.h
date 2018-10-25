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
  struct ether_hdr ethhdr; // TX Ethernet header
} EthFacePriv;

static uint16_t
EthFace_SizeofTxHeader()
{
  return sizeof(struct ether_hdr) + PrependLpHeader_GetHeadroom();
}

/** \brief Initialize a face to communicate on Ethernet.
 *  \param mempools headerMp must have \c EthFace_SizeofTxHeader() dataroom
 */
bool EthFace_Init(Face* face, FaceMempools* mempools);

#endif // NDN_DPDK_IFACE_ETHFACE_ETH_FACE_H
