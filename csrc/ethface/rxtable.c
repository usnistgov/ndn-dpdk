#include "rxtable.h"
#include "face.h"

__attribute__((nonnull)) static bool
EthRxTable_Accept(EthRxTable* rxt, struct rte_mbuf* m)
{
  // RCU lock is inherited from RxLoop_Run
  EthFacePriv* priv;
  struct cds_hlist_node* pos;
  cds_hlist_for_each_entry_rcu (priv, pos, &rxt->head, rxtNode) {
    if (EthRxMatch_Match(&priv->rxMatch, m)) {
      m->port = priv->faceID;
      return true;
    }
  }
  return false;
}

uint16_t
EthRxTable_RxBurst(RxGroup* rxg, struct rte_mbuf** pkts, uint16_t nPkts)
{
  EthRxTable* rxt = (EthRxTable*)rxg;
  uint16_t nInput = rte_eth_rx_burst(rxt->port, rxt->queue, pkts, nPkts);
  uint64_t now = rte_get_tsc_cycles();

  uint16_t nRx = 0, nRej = 0;
  struct rte_mbuf* rejects[MaxBurstSize];
  for (uint16_t i = 0; i < nInput; ++i) {
    struct rte_mbuf* m = pkts[i];
    if (likely(EthRxTable_Accept(rxt, m))) {
      m->timestamp = now;
      pkts[nRx++] = m;
    } else {
      rejects[nRej++] = m;
    }
  }

  if (unlikely(nRej > 0)) {
    rte_pktmbuf_free_bulk(rejects, nRej);
  }
  return nRx;
}
