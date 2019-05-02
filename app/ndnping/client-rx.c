#include "client-rx.h"

#include "../../core/logger.h"
#include "../../ndn/packet.h"
#include "token.h"

INIT_ZF_LOG(PingClient);

static bool
PingClientRx_GetSeqNumFromName(PingClientRx* cr,
                               const PingClientRxPattern* pattern,
                               const Name* name,
                               uint64_t* seqNum)
{
  if (unlikely(name->p.nOctets < pattern->prefixLen + PINGCLIENT_SUFFIX_LEN)) {
    return false;
  }

  const uint8_t* comp = RTE_PTR_ADD(name->v, pattern->prefixLen);
  if (unlikely(comp[0] != TT_GenericNameComponent ||
               comp[1] != sizeof(uint64_t))) {
    return false;
  }

  *seqNum = *(const unaligned_uint64_t*)RTE_PTR_ADD(comp, 2);
  return true;
}

static void
PingClientRx_ProcessData(PingClientRx* cr, Packet* npkt, uint64_t now)
{
  uint64_t token = Packet_GetLpL3Hdr(npkt)->pitToken;
  uint8_t patternId = NdnpingToken_GetPatternId(token);
  PingClientRxPattern* pattern = &cr->pattern[patternId];

  const PData* data = Packet_GetDataHdr(npkt);
  uint64_t seqNum;
  if (unlikely(
        NdnpingToken_GetRunNum(token) != cr->runNum ||
        patternId >= cr->nPatterns ||
        !PingClientRx_GetSeqNumFromName(cr, pattern, &data->name, &seqNum) ||
        PINGCLIENT_SELECT_PATTERN(cr, seqNum) != patternId)) {
    return;
  }

  ZF_LOGD(">D seq=%" PRIx64 " pattern=%d", seqNum, patternId);

  ++pattern->nData;
  uint64_t sendTime = NdnpingToken_GetTimestamp(token);
  RunningStat_Push(&pattern->rtt, now - sendTime);
}

static void
PingClientRx_ProcessNack(PingClientRx* cr, Packet* npkt, uint64_t now)
{
  uint64_t token = Packet_GetLpL3Hdr(npkt)->pitToken;
  uint8_t patternId = NdnpingToken_GetPatternId(token);
  PingClientRxPattern* pattern = &cr->pattern[patternId];

  const PNack* nack = Packet_GetNackHdr(npkt);
  uint64_t seqNum;
  if (unlikely(NdnpingToken_GetRunNum(token) != cr->runNum ||
               patternId >= cr->nPatterns ||
               !PingClientRx_GetSeqNumFromName(
                 cr, pattern, &nack->interest.name, &seqNum) ||
               PINGCLIENT_SELECT_PATTERN(cr, seqNum) != patternId)) {
    return;
  }

  ZF_LOGD(">N seq=%" PRIx64 " pattern=%d", seqNum, patternId);

  ++pattern->nNacks;
}

void
PingClientRx_Run(PingClientRx* cr)
{
  Packet* npkts[PINGCLIENT_RX_BURST_SIZE];

  while (ThreadStopFlag_ShouldContinue(&cr->stop)) {
    uint16_t nRx = rte_ring_sc_dequeue_bulk(
      cr->rxQueue, (void**)npkts, PINGCLIENT_RX_BURST_SIZE, NULL);
    uint64_t now = Ndnping_Now();
    for (uint16_t i = 0; i < nRx; ++i) {
      Packet* npkt = npkts[i];
      if (unlikely(Packet_GetL2PktType(npkt) != L2PktType_NdnlpV2)) {
        continue;
      }
      switch (Packet_GetL3PktType(npkt)) {
        case L3PktType_Data:
          PingClientRx_ProcessData(cr, npkt, now);
          break;
        case L3PktType_Nack:
          PingClientRx_ProcessNack(cr, npkt, now);
          break;
        default:
          assert(false);
          break;
      }
    }
    FreeMbufs((struct rte_mbuf**)npkts, nRx);
  }
}
