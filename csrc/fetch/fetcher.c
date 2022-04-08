#include "fetcher.h"

#include "../core/logger.h"
#include "../ndni/nni.h"

N_LOG_INIT(FetchProc);

__attribute__((nonnull)) static void
FetchProc_Encode(FetchProc* fp, FetchThread* fth, struct rte_mbuf* pkt, uint64_t segNum)
{
  uint8_t suffix[10];
  suffix[0] = TtSegmentNameComponent;
  suffix[1] = Nni_Encode(RTE_PTR_ADD(suffix, 2), segNum);
  LName nameSuffix = { .length = suffix[1] + 2, .value = suffix };

  uint32_t nonce = NonceGen_Next(&fth->nonceGen);
  Packet* npkt = InterestTemplate_Encode(&fp->tpl, pkt, nameSuffix, nonce);
  LpPitToken_Set(&Packet_GetLpL3Hdr(npkt)->pitToken, sizeof(fp->pitToken), &fp->pitToken);
}

__attribute__((nonnull)) static uint32_t
FetchProc_TxBurst(FetchProc* fp, FetchThread* fth)
{
  uint64_t segNums[MaxBurstSize];
  size_t count = FetchLogic_TxInterestBurst(&fp->logic, segNums, RTE_DIM(segNums));
  if (unlikely(count == 0)) {
    return count;
  }

  struct rte_mbuf* pkts[MaxBurstSize];
  int res = rte_pktmbuf_alloc_bulk(fth->interestMp, pkts, count);
  if (unlikely(res != 0)) {
    N_LOGW("%p interestMp-full", fp);
    return count;
  }

  for (size_t i = 0; i < count; ++i) {
    FetchProc_Encode(fp, fth, pkts[i], segNums[i]);
  }
  Face_TxBurst(fth->face, (Packet**)pkts, count);
  return count;
}

__attribute__((nonnull)) static bool
FetchProc_Decode(FetchProc* fp, Packet* npkt, FetchLogicRxData* lpkt)
{
  if (unlikely(Packet_GetType(npkt) != PktData)) {
    return false;
  }
  LpL3* lpl3 = Packet_GetLpL3Hdr(npkt);
  lpkt->congMark = lpl3->congMark;

  const PData* data = Packet_GetDataHdr(npkt);
  const uint8_t* seqNumComp = RTE_PTR_ADD(data->name.value, fp->tpl.prefixL);
  return data->name.length > fp->tpl.prefixL + 1 &&
         memcmp(data->name.value, fp->tpl.prefixV, fp->tpl.prefixL + 1) == 0 &&
         Nni_Decode(seqNumComp[1], RTE_PTR_ADD(seqNumComp, 2), &lpkt->segNum);
}

__attribute__((nonnull)) static uint32_t
FetchProc_RxBurst(FetchProc* fp)
{
  TscTime now = rte_get_tsc_cycles();
  Packet* npkts[MaxBurstSize];
  uint32_t nRx = PktQueue_Pop(&fp->rxQueue, (struct rte_mbuf**)npkts, MaxBurstSize, now).count;

  FetchLogicRxData lpkts[MaxBurstSize];
  size_t count = 0;
  for (uint16_t i = 0; i < nRx; ++i) {
    bool ok = FetchProc_Decode(fp, npkts[i], &lpkts[count]);
    if (likely(ok)) {
      ++count;
    }
  }
  FetchLogic_RxDataBurst(&fp->logic, lpkts, count);
  rte_pktmbuf_free_bulk((struct rte_mbuf**)npkts, nRx);
  return nRx;
}

int
FetchThread_Run(FetchThread* fth)
{
  uint32_t nProcessed = 0;
  while (ThreadCtrl_Continue(fth->ctrl, nProcessed)) {
    rcu_quiescent_state();
    rcu_read_lock();

    FetchProc* fp;
    struct cds_hlist_node* pos;
    cds_hlist_for_each_entry_rcu (fp, pos, &fth->head, fthNode) {
      MinSched_Trigger(fp->logic.sched);
      nProcessed += FetchProc_TxBurst(fp, fth);
      nProcessed += FetchProc_RxBurst(fp);
    }
    rcu_read_unlock();
  }
  return 0;
}
