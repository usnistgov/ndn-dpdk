#include "rx.h"

#include "../core/logger.h"
#include "../iface/face.h"
#include "token.h"

N_LOG_INIT(TgConsumer);

__attribute__((nonnull)) static bool
TgConsumerRx_GetSeqNumFromName(TgConsumerRx* cr, const TgConsumerRxPattern* pattern,
                               const PName* name, uint64_t* seqNum)
{
  if (unlikely(name->length < pattern->prefixLen + TGCONSUMER_SUFFIX_LEN)) {
    return false;
  }

  const uint8_t* comp = RTE_PTR_ADD(name->value, pattern->prefixLen);
  if (unlikely(comp[0] != TtGenericNameComponent || comp[1] != sizeof(uint64_t))) {
    return false;
  }

  *seqNum = *(const unaligned_uint64_t*)RTE_PTR_ADD(comp, 2);
  return true;
}

__attribute__((nonnull)) static void
TgConsumerRx_ProcessData(TgConsumerRx* cr, Packet* npkt)
{
  uint64_t token = Packet_GetLpL3Hdr(npkt)->pitToken;
  uint8_t patternId = TgToken_GetPatternId(token);
  TgConsumerRxPattern* pattern = &cr->pattern[patternId];

  const PData* data = Packet_GetDataHdr(npkt);
  uint64_t seqNum;
  if (unlikely(TgToken_GetRunNum(token) != cr->runNum || patternId >= cr->nPatterns ||
               !TgConsumerRx_GetSeqNumFromName(cr, pattern, &data->name, &seqNum))) {
    return;
  }

  N_LOGD(">D seq=%" PRIx64 " pattern=%d", seqNum, patternId);
  ++pattern->nData;
  TgTime recvTime = TgTime_FromTsc(Mbuf_GetTimestamp(Packet_ToMbuf(npkt)));
  TgTime sendTime = TgToken_GetTimestamp(token);
  RunningStat_Push(&pattern->rtt, recvTime - sendTime);
}

__attribute__((nonnull)) static void
TgConsumerRx_ProcessNack(TgConsumerRx* cr, Packet* npkt)
{
  uint64_t token = Packet_GetLpL3Hdr(npkt)->pitToken;
  uint8_t patternId = TgToken_GetPatternId(token);
  TgConsumerRxPattern* pattern = &cr->pattern[patternId];

  const PNack* nack = Packet_GetNackHdr(npkt);
  uint64_t seqNum;
  if (unlikely(TgToken_GetRunNum(token) != cr->runNum || patternId >= cr->nPatterns ||
               !TgConsumerRx_GetSeqNumFromName(cr, pattern, &nack->interest.name, &seqNum))) {
    return;
  }

  N_LOGD(">N seq=%" PRIx64 " pattern=%d", seqNum, patternId);
  ++pattern->nNacks;
}

int
TgConsumerRx_Run(TgConsumerRx* cr)
{
  struct rte_mbuf* pkts[MaxBurstSize];
  while (ThreadStopFlag_ShouldContinue(&cr->stop)) {
    TscTime now = rte_get_tsc_cycles();
    PktQueuePopResult pop = PktQueue_Pop(&cr->rxQueue, pkts, RTE_DIM(pkts), now);
    for (uint16_t i = 0; i < pop.count; ++i) {
      Packet* npkt = Packet_FromMbuf(pkts[i]);
      switch (Packet_GetType(npkt)) {
        case PktData:
          TgConsumerRx_ProcessData(cr, npkt);
          break;
        case PktNack:
          TgConsumerRx_ProcessNack(cr, npkt);
          break;
        default:
          NDNDPDK_ASSERT(false);
          break;
      }
    }
    rte_pktmbuf_free_bulk(pkts, pop.count);
  }
  return 0;
}
