#include "fetcher.h"

#include "../../core/logger.h"
#include "../../ndn/nni.h"

INIT_ZF_LOG(Fetcher);

#define FETCHER_TX_BURST_SIZE 64
#define FETCHER_RX_BURST_SIZE 64

static void
Fetcher_Encode(Fetcher* fetcher, Packet* npkt, uint64_t segNum)
{
  uint8_t suffix[10];
  suffix[0] = TT_SegmentNameComponent;
  suffix[1] = EncodeNni(&suffix[2], segNum);
  LName nameSuffix = { .length = suffix[1] + 2, .value = suffix };

  struct rte_mbuf* pkt = Packet_ToMbuf(npkt);
  pkt->data_off = fetcher->interestMbufHeadroom;
  EncodeInterest(pkt,
                 &fetcher->tpl,
                 fetcher->tplPrepareBuffer,
                 nameSuffix,
                 NonceGen_Next(&fetcher->nonceGen),
                 0,
                 NULL);

  Packet_SetL3PktType(npkt, L3PktType_Interest); // for stats; no PInterest*
}

static void
Fetcher_TxBurst(Fetcher* fetcher)
{
  uint64_t segNums[FETCHER_TX_BURST_SIZE];
  size_t count =
    FetchLogic_TxInterestBurst(&fetcher->logic, segNums, FETCHER_TX_BURST_SIZE);

  Packet* npkts[FETCHER_TX_BURST_SIZE];
  int res = rte_pktmbuf_alloc_bulk(
    fetcher->interestMp, (struct rte_mbuf**)npkts, count);
  if (unlikely(res != 0)) {
    ZF_LOGW("interestMp-full");
    return;
  }

  for (int i = 0; i < count; ++i) {
    Fetcher_Encode(fetcher, npkts[i], segNums[i]);
  }
  Face_TxBurst(fetcher->face, npkts, count);
}

static bool
Fetcher_Decode(Fetcher* fetcher, Packet* npkt, uint64_t* segNum)
{
  const PData* data = Packet_GetDataHdr(npkt);
  LName* name = (LName*)&data->name;
  const uint8_t* comp =
    RTE_PTR_ADD(name->value, fetcher->tpl.namePrefix.length);
  return LName_Compare(fetcher->tpl.namePrefix, *name) == NAMECMP_LPREFIX &&
         comp[0] == TT_SegmentNameComponent &&
         DecodeNni(comp[1], &comp[2], segNum) == NdnError_OK;
}

static void
Fetcher_RxBurst(Fetcher* fetcher)
{
  Packet* npkts[FETCHER_RX_BURST_SIZE];
  uint16_t nRx = rte_ring_sc_dequeue_burst(
    fetcher->rxQueue, (void**)npkts, FETCHER_RX_BURST_SIZE, NULL);

  uint64_t segNums[FETCHER_RX_BURST_SIZE];
  size_t count = 0;
  for (uint16_t i = 0; i < nRx; ++i) {
    bool ok = Fetcher_Decode(fetcher, npkts[i], &segNums[count]);
    if (likely(ok)) {
      ++count;
    }
  }
  FetchLogic_RxDataBurst(&fetcher->logic, segNums, count);
  FreeMbufs((struct rte_mbuf**)npkts, nRx);
}

int
Fetcher_Run(Fetcher* fetcher)
{
  while (ThreadStopFlag_ShouldContinue(&fetcher->stop)) {
    if (unlikely(FetchLogic_Finished(&fetcher->logic))) {
      return FETCHER_COMPLETED;
    }
    MinSched_Trigger(fetcher->logic.sched);
    Fetcher_TxBurst(fetcher);
    Fetcher_RxBurst(fetcher);
  }
  return FETCHER_STOPPED;
}
