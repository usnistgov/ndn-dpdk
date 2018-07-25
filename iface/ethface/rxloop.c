#include "rxloop.h"

EthRxLoop*
EthRxLoop_New(int maxTasks, int numaSocket)
{
  assert(maxTasks > 0);

  EthRxLoop* rxl = (EthRxLoop*)rte_zmalloc_socket(
    "EthRxLoop", sizeof(EthRxLoop) + maxTasks * sizeof(EthRxTask), 0,
    numaSocket);
  rxl->callback = (const struct rte_eth_rxtx_callback**)rte_zmalloc_socket(
    "EthRxLoopCallbackArray", maxTasks * sizeof(struct rte_eth_rxtx_callback*),
    0, numaSocket);
  if (rxl->callback == NULL) {
    rte_free(rxl);
    return NULL;
  }

  rxl->maxTasks = maxTasks;
  return rxl;
}

void
EthRxLoop_Close(EthRxLoop* rxl)
{
  for (int i = 0; i < rxl->nTasks; ++i) {
    EthRxTask* rxt = &rxl->task[i];
    rte_eth_remove_rx_callback(rxt->port, rxt->queue, rxl->callback[i]);
  }
  rte_free(rxl->callback);
  rte_free(rxl);
}

static uint16_t
EthFace_RxCallback(uint16_t port, uint16_t queue, struct rte_mbuf** pkts,
                   uint16_t nPkts, uint16_t maxPkts, void* rxt0)
{
  // TODO offload timestamping to hardware where available
  uint64_t now = rte_get_tsc_cycles();
  for (uint16_t i = 0; i < nPkts; ++i) {
    pkts[i]->timestamp = now;
  }
  return nPkts;
}

int
EthRxLoop_AddTask(EthRxLoop* rxl, EthRxTask* task)
{
  EthRxTask* rxt = &rxl->task[rxl->nTasks];
  const struct rte_eth_rxtx_callback* cb =
    rte_eth_add_rx_callback(task->port, task->queue, &EthFace_RxCallback, rxt);
  if (cb == NULL) {
    return rte_errno;
  }
  rxl->callback[rxl->nTasks] = cb;

  rte_memcpy(rxt, task, sizeof(*rxt));
  ++rxl->nTasks;
  return 0;
}

static bool
EthRxLoop_StripEtherHdr(struct rte_mbuf* pkt)
{
  assert(pkt->data_len >= sizeof(struct ether_hdr));
  const struct ether_hdr* eth = rte_pktmbuf_mtod(pkt, const struct ether_hdr*);

  // TODO offload ethertype filtering to hardware where available
  if (unlikely(eth->ether_type != rte_cpu_to_be_16(NDN_ETHERTYPE))) {
    rte_pktmbuf_free(pkt);
    return false;
  }

  rte_pktmbuf_adj(pkt, sizeof(struct ether_hdr));
  return true;
}

void
EthRxLoop_Run(EthRxLoop* rxl, FaceRxBurst* burst, Face_RxCb cb, void* cbarg)
{
  struct rte_mbuf** pkts = FaceRxBurst_GetScratch(burst);
  uint16_t burstSize = burst->capacity;

  while (likely(!rxl->stop)) {
    for (int i = 0; i < rxl->nTasks; ++i) {
      EthRxTask* rxt = &rxl->task[i];
      uint16_t nInput =
        rte_eth_rx_burst(rxt->port, rxt->queue, pkts, burstSize);

      uint16_t nRx = 0;
      for (uint16_t i = 0; i < nInput; ++i) {
        struct rte_mbuf* pkt = pkts[i];
        if (likely(EthRxLoop_StripEtherHdr(pkt))) {
          pkts[nRx++] = pkt;
        }
      }

      FaceImpl_RxBurst(rxt->face, rxt->rxThread, burst, nRx, cb, cbarg);
    }
  }
}
