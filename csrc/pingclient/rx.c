#include "rx.h"

#include "../core/logger.h"
#include "../iface/face.h"
#include "token.h"

INIT_ZF_LOG(PingClient);

__attribute__((nonnull)) static bool
PingClientRx_GetSeqNumFromName(PingClientRx* cr, const PingClientRxPattern* pattern,
                               const PName* name, uint64_t* seqNum)
{
  if (unlikely(name->length < pattern->prefixLen + PINGCLIENT_SUFFIX_LEN)) {
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
PingClientRx_ProcessData(PingClientRx* cr, Packet* npkt)
{
  uint64_t token = Packet_GetLpL3Hdr(npkt)->pitToken;
  uint8_t patternId = PingToken_GetPatternId(token);
  PingClientRxPattern* pattern = &cr->pattern[patternId];

  const PData* data = Packet_GetDataHdr(npkt);
  uint64_t seqNum;
  if (unlikely(PingToken_GetRunNum(token) != cr->runNum || patternId >= cr->nPatterns ||
               !PingClientRx_GetSeqNumFromName(cr, pattern, &data->name, &seqNum))) {
    return;
  }

  ZF_LOGD(">D seq=%" PRIx64 " pattern=%d", seqNum, patternId);
  ++pattern->nData;
  PingTime recvTime = PingTime_FromTsc(Packet_ToMbuf(npkt)->timestamp);
  PingTime sendTime = PingToken_GetTimestamp(token);
  RunningStat_Push(&pattern->rtt, recvTime - sendTime);
}

__attribute__((nonnull)) static void
PingClientRx_ProcessNack(PingClientRx* cr, Packet* npkt)
{
  uint64_t token = Packet_GetLpL3Hdr(npkt)->pitToken;
  uint8_t patternId = PingToken_GetPatternId(token);
  PingClientRxPattern* pattern = &cr->pattern[patternId];

  const PNack* nack = Packet_GetNackHdr(npkt);
  uint64_t seqNum;
  if (unlikely(PingToken_GetRunNum(token) != cr->runNum || patternId >= cr->nPatterns ||
               !PingClientRx_GetSeqNumFromName(cr, pattern, &nack->interest.name, &seqNum))) {
    return;
  }

  ZF_LOGD(">N seq=%" PRIx64 " pattern=%d", seqNum, patternId);
  ++pattern->nNacks;
}

int
PingClientRx_Run(PingClientRx* cr)
{
  struct rte_mbuf* pkts[MaxBurstSize];
  while (ThreadStopFlag_ShouldContinue(&cr->stop)) {
    TscTime now = rte_get_tsc_cycles();
    PktQueuePopResult pop = PktQueue_Pop(&cr->rxQueue, pkts, RTE_DIM(pkts), now);
    for (uint16_t i = 0; i < pop.count; ++i) {
      Packet* npkt = Packet_FromMbuf(pkts[i]);
      switch (Packet_GetType(npkt)) {
        case PktData:
          PingClientRx_ProcessData(cr, npkt);
          break;
        case PktNack:
          PingClientRx_ProcessNack(cr, npkt);
          break;
        default:
          assert(false);
          break;
      }
    }
    rte_pktmbuf_free_bulk_(pkts, pop.count);
  }
  return 0;
}
