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
  EthRxTable* rxt = container_of(rxg, EthRxTable, base);
  struct rte_mbuf* receives[MaxBurstSize];
  uint16_t nInput = rte_eth_rx_burst(rxt->port, rxt->queue, receives, nPkts);
  uint64_t now = rte_get_tsc_cycles();

  uint16_t nRx = 0, nUnmatch = 0, nDrop = 0;
  struct rte_mbuf* unmatch[MaxBurstSize];
  struct rte_mbuf* drop[MaxBurstSize];
  for (uint16_t i = 0; i < nInput; ++i) {
    struct rte_mbuf* m = receives[i];
    Mbuf_SetTimestamp(m, now);
    if (unlikely(!EthRxTable_Accept(rxt, m))) {
      unmatch[nUnmatch++] = m;
      continue;
    }

    if (likely(rxt->copyTo == NULL)) {
      pkts[nRx++] = m;
      continue;
    }

    struct rte_mbuf* copy = rte_pktmbuf_copy(m, rxt->copyTo, 0, UINT32_MAX);
    if (likely(copy != NULL)) {
      pkts[nRx++] = copy;
    }
    drop[nDrop++] = m;
  }

  if (unlikely(nUnmatch > 0)) {
    if (!PdumpSourceRef_Process(&rxt->pdumpUnmatched, unmatch, nUnmatch)) {
      rte_pktmbuf_free_bulk(unmatch, nUnmatch);
    }
  }
  if (unlikely(nDrop > 0)) {
    rte_pktmbuf_free_bulk(drop, nDrop);
  }
  return nRx;
}
