#include "fetcher.h"

#include "../core/logger.h"
#include "../ndn/nni.h"

INIT_ZF_LOG(FetchProc);

#define FETCHER_TX_BURST_SIZE 64

static void
FetchProc_Encode(FetchProc* fp, FetchThread* fth, Packet* npkt, uint64_t segNum)
{
  uint8_t suffix[10];
  suffix[0] = TtSegmentNameComponent;
  suffix[1] = EncodeNni(&suffix[2], segNum);
  LName nameSuffix = { .length = suffix[1] + 2, .value = suffix };

  struct rte_mbuf* pkt = Packet_ToMbuf(npkt);
  uint32_t nonce = NonceGen_Next(&fth->nonceGen);
  EncodeInterest(pkt, &fp->tpl, nameSuffix, nonce);

  Packet_SetL3PktType(npkt, L3PktTypeInterest); // for stats; no PInterest*
  Packet_InitLpL3Hdr(npkt)->pitToken = fp->pitToken;
}

static void
FetchProc_TxBurst(FetchProc* fp, FetchThread* fth)
{
  uint64_t segNums[FETCHER_TX_BURST_SIZE];
  size_t count =
    FetchLogic_TxInterestBurst(&fp->logic, segNums, FETCHER_TX_BURST_SIZE);
  if (unlikely(count == 0)) {
    return;
  }

  Packet* npkts[FETCHER_TX_BURST_SIZE];
  int res =
    rte_pktmbuf_alloc_bulk(fth->interestMp, (struct rte_mbuf**)npkts, count);
  if (unlikely(res != 0)) {
    ZF_LOGW("%p interestMp-full", fp);
    return;
  }

  for (size_t i = 0; i < count; ++i) {
    FetchProc_Encode(fp, fth, npkts[i], segNums[i]);
  }
  Face_TxBurst(fth->face, npkts, count);
}

static bool
FetchProc_Decode(FetchProc* fp, Packet* npkt, FetchLogicRxData* lpkt)
{
  if (unlikely(Packet_GetL3PktType(npkt) != L3PktTypeData)) {
    return false;
  }
  LpL3* lpl3 = Packet_GetLpL3Hdr(npkt);
  lpkt->congMark = lpl3->congMark;

  const PData* data = Packet_GetDataHdr(npkt);
  const uint8_t* seqNumComp = RTE_PTR_ADD(data->name.v, fp->tpl.prefixL);
  return data->name.p.nOctets > fp->tpl.prefixL + 1 &&
         memcmp(data->name.v, fp->tpl.prefixV, fp->tpl.prefixL + 1) == 0 &&
         DecodeNni(seqNumComp[1], &seqNumComp[2], &lpkt->segNum) == NdnErrOK;
}

static void
FetchProc_RxBurst(FetchProc* fp)
{
  Packet* npkts[PKTQUEUE_BURST_SIZE_MAX];
  uint32_t nRx = PktQueue_Pop(&fp->rxQueue,
                              (struct rte_mbuf**)npkts,
                              PKTQUEUE_BURST_SIZE_MAX,
                              rte_get_tsc_cycles())
                   .count;

  FetchLogicRxData lpkts[PKTQUEUE_BURST_SIZE_MAX];
  size_t count = 0;
  for (uint16_t i = 0; i < nRx; ++i) {
    bool ok = FetchProc_Decode(fp, npkts[i], &lpkts[count]);
    if (likely(ok)) {
      ++count;
    }
  }
  FetchLogic_RxDataBurst(&fp->logic, lpkts, count);
  FreeMbufs((struct rte_mbuf**)npkts, nRx);
}

void
FetchProc_Once(FetchProc* fp, FetchThread* fth)
{
  MinSched_Trigger(fp->logic.sched);
  FetchProc_TxBurst(fp, fth);
  FetchProc_RxBurst(fp);
}

int
FetchThread_Run(FetchThread* fth)
{
  while (ThreadStopFlag_ShouldContinue(&fth->stop)) {
    rcu_quiescent_state();
    rcu_read_lock();

    FetchProc* fp;
    struct cds_hlist_node* pos;
    cds_hlist_for_each_entry_rcu(fp, pos, &fth->head, fthNode)
    {
      FetchProc_Once(fp, fth);
    }
    rcu_read_unlock();
  }
  return 0;
}
