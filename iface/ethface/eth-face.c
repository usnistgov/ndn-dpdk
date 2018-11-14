#include "eth-face.h"
#include "../../core/logger.h"

INIT_ZF_LOG(EthFace);

// EthFace currently only supports one TX queue,
// so queue number is hardcoded with this macro.
#define QUEUE_0 0

uint16_t
EthFace_TxBurst(Face* face, struct rte_mbuf** pkts, uint16_t nPkts)
{
  EthFacePriv* priv = Face_GetPrivT(face, EthFacePriv);

  for (uint16_t i = 0; i < nPkts; ++i) {
    char* room = rte_pktmbuf_prepend(pkts[i], sizeof(priv->txHdr));
    assert(room != NULL); // enough headroom is required
    rte_memcpy(room, &priv->txHdr, sizeof(priv->txHdr));

    // XXX (1) FaceImpl_CountSent only wants transmitted packets, not tail-dropped packets.
    // XXX (2) FaceImpl_CountSent should be invoked after transmission, not before enqueuing.
    // XXX note: invoking FaceImpl_CountSent from txCallback has the same problems.
    FaceImpl_CountSent(face, pkts[i]);
  }
  return rte_eth_tx_burst(priv->port, QUEUE_0, pkts, nPkts);
}
