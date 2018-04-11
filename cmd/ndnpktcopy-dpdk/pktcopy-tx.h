#ifndef NDN_DPDK_CMD_NDNPKTCOPY_TX_H
#define NDN_DPDK_CMD_NDNPKTCOPY_TX_H

/// \file

#include "../../iface/iface.h"

/** \brief Transmitting thread of ndnpktcopy.
 */
typedef struct PktcopyTx
{
  FaceId face;
  struct rte_ring* txRing;
} PktcopyTx;

void PktcopyTx_Run(PktcopyTx* pctx);

#endif // NDN_DPDK_CMD_NDNPKTCOPY_TX_H
