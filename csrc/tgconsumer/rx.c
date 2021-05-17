#include "rx.h"

#include "../core/logger.h"
#include "../iface/face.h"

N_LOG_INIT(Tgc);

__attribute__((nonnull)) static bool
TgcRx_GetSeqNumFromName(TgcRx* cr, const TgcRxPattern* pattern, const PName* name, uint64_t* seqNum)
{
  if (unlikely(name->length < pattern->prefixLen + TGCONSUMER_SEQNUM_SIZE)) {
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
TgcRx_ProcessData(TgcRx* cr, Packet* npkt, uint8_t id, TscTime sendTime)
{
  TgcRxPattern* pattern = &cr->pattern[id];
  const PData* data = Packet_GetDataHdr(npkt);

  uint64_t seqNum;
  if (unlikely(!TgcRx_GetSeqNumFromName(cr, pattern, &data->name, &seqNum))) {
    return;
  }

  N_LOGD(">D seq=%" PRIx64 " pattern=%" PRIu8, seqNum, id);
  ++pattern->nData;
  TscTime recvTime = Mbuf_GetTimestamp(Packet_ToMbuf(npkt));
  RunningStat_Push(&pattern->rtt, recvTime - sendTime);
}

__attribute__((nonnull)) static void
TgcRx_ProcessNack(TgcRx* cr, Packet* npkt, uint8_t id)
{
  TgcRxPattern* pattern = &cr->pattern[id];
  const PNack* nack = Packet_GetNackHdr(npkt);

  uint64_t seqNum;
  if (unlikely(!TgcRx_GetSeqNumFromName(cr, pattern, &nack->interest.name, &seqNum))) {
    return;
  }

  N_LOGD(">N seq=%" PRIx64 " pattern=%" PRIu8, seqNum, id);
  ++pattern->nNacks;
}

int
TgcRx_Run(TgcRx* cr)
{
  struct rte_mbuf* pkts[MaxBurstSize];
  while (ThreadStopFlag_ShouldContinue(&cr->stop)) {
    TscTime now = rte_get_tsc_cycles();
    PktQueuePopResult pop = PktQueue_Pop(&cr->rxQueue, pkts, RTE_DIM(pkts), now);
    for (uint16_t i = 0; i < pop.count; ++i) {
      Packet* npkt = Packet_FromMbuf(pkts[i]);
      const LpPitToken* token = &Packet_GetLpL3Hdr(npkt)->pitToken;
      if (unlikely(token->length != TgcTokenLength) ||
          unlikely(TgcToken_GetRunNum(token) != cr->runNum)) {
        continue;
      }
      uint8_t id = TgcToken_GetPatternID(token);
      if (unlikely(id >= cr->nPatterns)) {
        continue;
      }
      switch (Packet_GetType(npkt)) {
        case PktData:
          TgcRx_ProcessData(cr, npkt, id, TgcToken_GetTimestamp(token));
          break;
        case PktNack:
          TgcRx_ProcessNack(cr, npkt, id);
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
