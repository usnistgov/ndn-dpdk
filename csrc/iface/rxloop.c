#include "rxloop.h"

RxGroup theChanRxGroup_;

__attribute__((nonnull)) static void
RxLoop_Transfer(RxLoop* rxl, RxGroup* rxg)
{
  struct rte_mbuf* frames[MaxBurstSize];
  uint16_t nRx = (*rxg->rxBurstOp)(rxg, frames, RTE_DIM(frames));
  for (uint16_t i = 0; i < nRx; ++i) {
    struct rte_mbuf* frame = frames[i];
    Face* face = Face_Get(frame->port);
    if (unlikely(face->impl == NULL)) {
      rte_pktmbuf_free(frame);
      continue;
    }

    Packet* npkt = RxProc_Input(&face->impl->rx, rxg->rxThread, frame);
    if (npkt == NULL) {
      continue;
    }

    switch (Packet_GetType(npkt)) {
      case PktInterest: {
        PInterest* interest = Packet_GetInterestHdr(npkt);
        InputDemux_Dispatch(&rxl->demuxI, npkt, &interest->name);
        break;
      }
      case PktData: {
        PData* data = Packet_GetDataHdr(npkt);
        InputDemux_Dispatch(&rxl->demuxD, npkt, &data->name);
        break;
      }
      case PktNack: {
        PNack* nack = Packet_GetNackHdr(npkt);
        InputDemux_Dispatch(&rxl->demuxN, npkt, &nack->interest.name);
        break;
      }
      default:
        assert(false);
        break;
    }
  }
}

int
RxLoop_Run(RxLoop* rxl)
{
  rcu_register_thread();
  while (ThreadStopFlag_ShouldContinue(&rxl->stop)) {
    rcu_quiescent_state();
    rcu_read_lock();

    RxGroup* rxg;
    struct cds_hlist_node* pos;
    cds_hlist_for_each_entry_rcu (rxg, pos, &rxl->head, rxlNode) {
      RxLoop_Transfer(rxl, rxg);
    }
    rcu_read_unlock();
  }
  rcu_unregister_thread();
  return 0;
}
