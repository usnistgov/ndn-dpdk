#include "rxloop.h"

EthRxLoop*
EthRxLoop_New(int maxTasks, int numaSocket)
{
  assert(maxTasks > 0);
  EthRxLoop* rxl = (EthRxLoop*)rte_zmalloc_socket(
    "EthRxLoop", sizeof(EthRxLoop) + maxTasks * sizeof(EthRxTask), 0,
    numaSocket);
  rxl->maxTasks = maxTasks;
  return rxl;
}

void
EthRxLoop_Close(EthRxLoop* rxl)
{
  rte_free(rxl);
}

void
EthRxLoop_AddTask(EthRxLoop* rxl, EthRxTask* task)
{
  EthRxTask* rxt = &rxl->task[rxl->nTasks];
  rte_memcpy(rxt, task, sizeof(*rxt));
  ++rxl->nTasks;
}

static bool
EthRxLoop_StripEtherHdr(struct rte_mbuf* frame)
{
  assert(frame->data_len >= sizeof(struct ether_hdr));
  const struct ether_hdr* eth =
    rte_pktmbuf_mtod(frame, const struct ether_hdr*);

  // TODO offload ethertype filtering to hardware where available
  if (unlikely(eth->ether_type != rte_cpu_to_be_16(NDN_ETHERTYPE))) {
    rte_pktmbuf_free(frame);
    return false;
  }

  rte_pktmbuf_adj(frame, sizeof(struct ether_hdr));
  return true;
}

void
EthRxLoop_Run(EthRxLoop* rxl, FaceRxBurst* burst, Face_RxCb cb, void* cbarg)
{
  struct rte_mbuf** frames = FaceRxBurst_GetScratch(burst);
  uint16_t burstSize = burst->capacity;

  while (likely(!rxl->stop)) {
    for (int i = 0; i < rxl->nTasks; ++i) {
      EthRxTask* rxt = &rxl->task[i];
      uint16_t nInput =
        rte_eth_rx_burst(rxt->port, rxt->queue, frames, burstSize);

      uint64_t now = rte_get_tsc_cycles();
      uint16_t nRx = 0;
      for (uint16_t i = 0; i < nInput; ++i) {
        struct rte_mbuf* frame = frames[i];
        frame->port = rxt->face->id;

        // TODO offload timestamping to hardware where available
        frame->timestamp = now;

        if (likely(EthRxLoop_StripEtherHdr(frame))) {
          frames[nRx++] = frame;
        }
      }

      FaceImpl_RxBurst(burst, nRx, rxt->rxThread, cb, cbarg);
    }
  }
}
