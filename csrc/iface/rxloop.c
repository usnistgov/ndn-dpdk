#include "rxloop.h"
#include "face-impl.h"

__attribute__((nonnull)) static uint16_t
RxLoop_Transfer(RxLoop* rxl, RxGroup* rxg) {
  RxGroupBurstCtx ctx;
  memset(&ctx, 0, offsetof(RxGroupBurstCtx, zeroizeEnd_));
  rxg->rxBurst(rxg, &ctx);

  struct rte_mbuf* drops[MaxBurstSize];
  uint16_t nDrops = 0;
  for (uint16_t i = 0; i < ctx.nRx; ++i) {
    struct rte_mbuf* pkt = ctx.pkts[i];

    bool dropped = (ctx.dropBits[i >> 6] & (1 << (i & 0x3F))) != 0;
    if (unlikely(dropped)) {
      if (likely(pkt != NULL)) {
        drops[nDrops++] = pkt;
      } else {
        // pkt was passed to pdump or freed as bounceBufs in EthRxTable_RxBurst
      }
      continue;
    }

    Face* face = Face_Get(pkt->port);
    if (unlikely(face->impl == NULL)) {
      drops[nDrops++] = pkt;
      continue;
    }

    PdumpSourceRef_Process(&face->impl->rxPdump, &pkt, 1);

    Packet* npkt = FaceRx_Input(face, rxg->rxThread, pkt);
    NULLize(pkt);
    if (npkt == NULL) {
      continue;
    }

    InputDemuxes* demuxes =
      likely(face->impl->rxDemuxes == NULL) ? &rxl->demuxes : face->impl->rxDemuxes;
    bool accepted = InputDemux_Dispatch(InputDemux_Of(demuxes, Packet_GetType(npkt)), npkt);
    if (unlikely(!accepted)) {
      drops[nDrops++] = Packet_ToMbuf(npkt);
    }
  }

  if (unlikely(nDrops > 0)) {
    rte_pktmbuf_free_bulk(drops, nDrops);
  }
  return ctx.nRx;
}

int
RxLoop_Run(RxLoop* rxl) {
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
