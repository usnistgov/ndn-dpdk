#ifndef NDN_DPDK_CMD_NDNPKTCOPY_RX_H
#define NDN_DPDK_CMD_NDNPKTCOPY_RX_H

/// \file

#include "../../iface/face.h"

/** \brief Maximum number of TX rings.
 */
#define PKTCOPYRX_MAXTX 16

/** \brief Receiving thread of ndnpktcopy.
 */
typedef struct PktcopyRx
{
  struct rte_mempool* indirectMp;
  struct rte_ring* txRings[PKTCOPYRX_MAXTX];
  int nTxRings;

  uint64_t nAllocError;
  uint64_t nTxRingCongestions[PKTCOPYRX_MAXTX];
} PktcopyRx;

void PktcopyRx_AddTxRing(PktcopyRx* pcrx, struct rte_ring* r);

void PktcopyRx_Rx(Face* face, FaceRxBurst* burst, void* pcrx0);

#endif // NDN_DPDK_CMD_NDNPKTCOPY_RX_H
