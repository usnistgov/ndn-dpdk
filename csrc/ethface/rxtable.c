#include "rxtable.h"
#include "face.h"

__attribute__((nonnull)) static bool
EthRxTable_Accept(EthRxTable* rxt, struct rte_mbuf* m, uint64_t now)
{
  // RCU lock is inherited from RxLoop_Run

  m->timestamp = now;

  EthFacePriv* priv;
  struct cds_hlist_node* pos;
  cds_hlist_for_each_entry_rcu (priv, pos, &rxt->head, rxtNode) {
    if (EthRxMatch_Match(&priv->rxMatch, m)) {
      m->port = priv->faceID;
      return true;
    }
  }

  rte_pktmbuf_free(m);
  return false;
}

uint16_t
EthRxTable_RxBurst(RxGroup* rxg, struct rte_mbuf** pkts, uint16_t nPkts)
{
  EthRxTable* rxt = (EthRxTable*)rxg;
  uint16_t nInput = rte_eth_rx_burst(rxt->port, rxt->queue, pkts, nPkts);
  uint64_t now = rte_get_tsc_cycles();
  uint16_t nRx = 0;
  for (uint16_t i = 0; i < nInput; ++i) {
    struct rte_mbuf* frame = pkts[i];
    if (likely(EthRxTable_Accept(rxt, frame, now))) {
      pkts[nRx++] = frame;
    }
  }
  return nRx;
}
