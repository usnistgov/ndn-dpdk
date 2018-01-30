#ifndef NDN_DPDK_CMD_NDNPKTCOPY_TX_H
#define NDN_DPDK_CMD_NDNPKTCOPY_TX_H

#include "../../iface/face.h"

/// \file

/** \brief Transmitting thread of ndnpktcopy.
 */
typedef struct PktcopyTx
{
  Face* face;
  struct rte_ring* txRing;
} PktcopyTx;

void PktcopyTx_Run(PktcopyTx* pctx);

#endif // NDN_DPDK_CMD_NDNPKTCOPY_TX_H
