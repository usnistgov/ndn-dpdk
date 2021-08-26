#include "rxloop.h"

RxGroup theChanRxGroup_;

__attribute__((nonnull)) static uint16_t
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
    RxProc* rx = &face->impl->rx;
    PdumpFaceRef_Process(&rx->pdump, face->id, &frame, 1);
    Packet* npkt = RxProc_Input(rx, rxg->rxThread, frame);
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
        NDNDPDK_ASSERT(false);
        break;
    }
  }

  return nRx;
}

int
RxLoop_Run(RxLoop* rxl)
{
  rcu_register_thread();
  uint16_t nProcessed = 0;
  while (ThreadCtrl_Continue(rxl->ctrl, nProcessed)) {
    rcu_quiescent_state();
    rcu_read_lock();
    RxGroup* rxg;
    struct cds_hlist_node* pos;
    cds_hlist_for_each_entry_rcu (rxg, pos, &rxl->head, rxlNode) {
      nProcessed += RxLoop_Transfer(rxl, rxg);
    }
    rcu_read_unlock();
  }
  rcu_unregister_thread();
  return 0;
}
