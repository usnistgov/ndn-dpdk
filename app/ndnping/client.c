#include "client.h"
#include "token.h"

#include "../../core/logger.h"

#include <rte_random.h>

INIT_ZF_LOG(NdnpingClient);

void
NdnpingClient_Init(NdnpingClient* client)
{
  client->suffixComponent.compT = TT_GenericNameComponent;
  client->suffixComponent.compL = 8;
  client->suffixComponent.compV = rte_rand();

  client->interestTpl.canBePrefix = true;
  client->interestTpl.mustBeFresh = true;
  client->interestTpl.fhL = 0;
  client->interestTpl.fhV = NULL;
  client->interestTpl.lifetime = client->interestLifetime;
  client->interestTpl.hopLimit = 255;

  uint16_t res = InterestTemplate_Prepare(
    &client->interestTpl, client->interestPrepareBuffer,
    sizeof(client->interestPrepareBuffer));
  assert(res == 0);
  NonceGen_Init(&client->nonceGen);
}

static uint8_t
NdnpingClient_SelectPattern(NdnpingClient* client, uint64_t seqNum)
{
  return seqNum % client->patterns.nRecords;
}

static void
NdnpingClient_PrepareTxInterest(NdnpingClient* client, Packet* npkt,
                                uint64_t now)
{
  struct rte_mbuf* pkt = Packet_ToMbuf(npkt);
  pkt->data_off = client->interestMbufHeadroom;
  uint64_t seqNum = ++client->suffixComponent.compV;
  uint8_t patternId = NdnpingClient_SelectPattern(client, seqNum);
  NdnpingClientPattern* pattern =
    NameSet_GetUsrT(&client->patterns, patternId, NdnpingClientPattern*);
  ++pattern->nInterests;

  client->interestTpl.namePrefix =
    NameSet_GetName(&client->patterns, patternId);
  LName nameSuffix = {.length = 10, .value = &client->suffixComponent.compT };
  EncodeInterest(pkt, &client->interestTpl, client->interestPrepareBuffer,
                 nameSuffix, NonceGen_Next(&client->nonceGen), 0, NULL);
  Packet_SetL3PktType(npkt, L3PktType_Interest); // for stats; no PInterest*
  ZF_LOGD("<I seq=%" PRIx64 " pattern=%d", seqNum, patternId);

  Packet_InitLpL3Hdr(npkt)->pitToken = NdnpingToken_New(patternId, now);
}

static void
NdnpingClient_TxBurst(NdnpingClient* client)
{
  Packet* npkts[NDNPINGCLIENT_TX_BURST_SIZE];
  int res = rte_pktmbuf_alloc_bulk(client->interestMp, (struct rte_mbuf**)npkts,
                                   NDNPINGCLIENT_TX_BURST_SIZE);
  if (unlikely(res != 0)) {
    ZF_LOGW("interestMp-full");
    return;
  }

  uint64_t now = Ndnping_Now();
  for (uint16_t i = 0; i < NDNPINGCLIENT_TX_BURST_SIZE; ++i) {
    NdnpingClient_PrepareTxInterest(client, npkts[i], now);
  }
  Face_TxBurst(client->face, npkts, NDNPINGCLIENT_TX_BURST_SIZE);
}

void
NdnpingClient_RunTx(NdnpingClient* client)
{
  TscTime nextTxBurst = rte_get_tsc_cycles();
  while (ThreadStopFlag_ShouldContinue(&client->txStop)) {
    if (rte_get_tsc_cycles() < nextTxBurst) {
      rte_pause();
      continue;
    }
    NdnpingClient_TxBurst(client);
    nextTxBurst += client->burstInterval;
  }
}

static bool
NdnpingClient_GetSeqNumFromName(NdnpingClient* client, uint8_t patternId,
                                const Name* name, uint64_t* seqNum)
{
  LName prefix = NameSet_GetName(&client->patterns, patternId);
  if (unlikely(name->p.nOctets < prefix.length + 10)) {
    return false;
  }

  const uint8_t* comp = RTE_PTR_ADD(name->v, prefix.length);
  if (unlikely(comp[0] != TT_GenericNameComponent || comp[1] != 8)) {
    return false;
  }

  *seqNum = *(const unaligned_uint64_t*)RTE_PTR_ADD(comp, 2);
  return true;
}

static void
NdnpingClient_ProcessRxData(NdnpingClient* client, Packet* npkt, uint64_t now)
{
  uint64_t token = Packet_GetLpL3Hdr(npkt)->pitToken;
  uint8_t patternId = NdnpingToken_GetPatternId(token);

  const PData* data = Packet_GetDataHdr(npkt);
  uint64_t seqNum;
  if (unlikely(!NdnpingClient_GetSeqNumFromName(client, patternId, &data->name,
                                                &seqNum) ||
               NdnpingClient_SelectPattern(client, seqNum) != patternId)) {
    return;
  }

  ZF_LOGD(">D seq=%" PRIx64 " pattern=%d", seqNum, patternId);

  NdnpingClientPattern* pattern =
    NameSet_GetUsrT(&client->patterns, patternId, NdnpingClientPattern*);
  ++pattern->nData;

  uint64_t sendTime = NdnpingToken_GetTimestamp(token);
  RunningStat_Push(&pattern->rtt, now - sendTime);
}

static void
NdnpingClient_ProcessRxNack(NdnpingClient* client, Packet* npkt, uint64_t now)
{
  uint64_t token = Packet_GetLpL3Hdr(npkt)->pitToken;
  uint8_t patternId = NdnpingToken_GetPatternId(token);

  const PNack* nack = Packet_GetNackHdr(npkt);
  uint64_t seqNum;
  if (unlikely(!NdnpingClient_GetSeqNumFromName(
                 client, patternId, &nack->interest.name, &seqNum) ||
               NdnpingClient_SelectPattern(client, seqNum) != patternId)) {
    return;
  }

  ZF_LOGD(">D seq=%" PRIx64 " pattern=%d", seqNum, patternId);

  NdnpingClientPattern* pattern =
    NameSet_GetUsrT(&client->patterns, patternId, NdnpingClientPattern*);
  ++pattern->nNacks;
}

void
NdnpingClient_RunRx(NdnpingClient* client)
{
  const int burstSize = 64;
  Packet* rx[burstSize];

  while (ThreadStopFlag_ShouldContinue(&client->rxStop)) {
    uint16_t nRx =
      rte_ring_sc_dequeue_bulk(client->rxQueue, (void**)rx, burstSize, NULL);
    uint64_t now = Ndnping_Now();
    for (uint16_t i = 0; i < nRx; ++i) {
      Packet* npkt = rx[i];
      if (unlikely(Packet_GetL2PktType(npkt) != L2PktType_NdnlpV2)) {
        continue;
      }
      switch (Packet_GetL3PktType(npkt)) {
        case L3PktType_Data:
          NdnpingClient_ProcessRxData(client, npkt, now);
          break;
        case L3PktType_Nack:
          NdnpingClient_ProcessRxNack(client, npkt, now);
          break;
        default:
          assert(false);
          break;
      }
    }
    FreeMbufs((struct rte_mbuf**)rx, nRx);
  }
}
