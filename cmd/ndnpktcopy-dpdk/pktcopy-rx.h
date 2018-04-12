#ifndef NDN_DPDK_CMD_NDNPKTCOPY_RX_H
#define NDN_DPDK_CMD_NDNPKTCOPY_RX_H

/// \file

#include "../../iface/iface.h"

#define PKTCOPYRX_RXBURST_SIZE 64
#define PKTCOPYRX_MAXTX 16

/** \brief Receiving thread of ndnpktcopy.
 */
typedef struct PktcopyRx
{
  struct rte_mempool* headerMp;
  struct rte_mempool* indirectMp;
  struct rte_ring* dumpRing;
  uint64_t nAllocError;
  int nTxFaces;
  FaceId txFaces[PKTCOPYRX_MAXTX];
} PktcopyRx;

void PktcopyRx_Rx(FaceId faceId, FaceRxBurst* burst, void* pcrx0);

#endif // NDN_DPDK_CMD_NDNPKTCOPY_RX_H
