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
  bool stopRxLoop;
  struct ether_hdr ethhdr; // TX Ethernet header
  void* rxCallback;
  void* txCallback;
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

void EthFace_Close(Face* face);

/** \brief Continually retrieve packets from an Ethernet face.
 *  \param burstSize how many L2 frames to retrieve in each burst.
 *  \param cb callback after each packet arrival.
 */
void EthFace_RxLoop(Face* face, uint16_t burstSize, Face_RxCb cb, void* cbarg);

#endif // NDN_DPDK_IFACE_ETHFACE_ETH_FACE_H
