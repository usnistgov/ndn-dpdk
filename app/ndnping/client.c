#include "client.h"

#include "../../core/logger.h"

#include <rte_cycles.h>
#include <rte_random.h>

INIT_ZF_LOG(NdnpingClient);

#define NDNPINGCLIENT_RX_BURST_SIZE 64
#define NDNPINGCLIENT_TX_BURST_SIZE 64
#define NDNPINGCLIENT_INTEREST_LIFETIME 1000

// Currently, only pattern 0 has timeout and RTT sampling, because seqNo determines which pattern
// to use, as well as whether to sample. This should be fixed when implementing pattern ratios.
#define PATTERN_0 0

typedef struct NdnpingClientSample
{
  bool isPending : 1;
  bool _reserved : 1;
  int patternId : 6;
  uint64_t sendTime : 56; ///< TSC cycles >> NDNPING_TIMING_PRECISION
} __rte_packed NdnpingClientSample;
static_assert(sizeof(NdnpingClientSample) == sizeof(uint64_t), "");

static bool
NdnpingClient_IsSamplingEnabled(NdnpingClient* client)
{
  return client->sampleFreq != 0xFF;
}

void
NdnpingClient_Init(NdnpingClient* client)
{
  static_assert(sizeof(client->suffixComponent) == 16, "");
  client->suffixComponent.compT = TT_GenericNameComponent;
  client->suffixComponent.compL = 8;
  client->suffixComponent.compV = rte_rand();

  client->interestTpl.nameSuffixSize = 10;
  client->interestTpl.nameSuffix = &client->suffixComponent.compT;
  client->interestTpl.mustBeFresh = true;
  client->interestTpl.lifetime = NDNPINGCLIENT_INTEREST_LIFETIME;
  client->interestTpl.fwHints = NULL;
  client->interestTpl.fwHintsSize = 0;

  if (NdnpingClient_IsSamplingEnabled(client)) {
    size_t nSampleTableEntries = 1 << client->sampleTableSize;
    client->samplingMask = (1 << client->sampleFreq) - 1;
    client->sampleIndexMask = (1 << client->sampleTableSize) - 1;
    client->sampleTable = rte_calloc_socket(
      "NdnpingClient.sampleTable", nSampleTableEntries,
      sizeof(NdnpingClientSample), 0, Face_GetNumaSocket(client->face));
  }
}

void
NdnpingClient_Close(NdnpingClient* client)
{
  if (client->sampleTable != NULL) {
    rte_free(client->sampleTable);
  }
}

static int
NdnpingClient_SelectPattern(NdnpingClient* client, uint64_t seqNo)
{
  return seqNo % client->patterns.nRecords;
}

static NdnpingClientSample*
NdnpingClient_FindSample(NdnpingClient* client, uint64_t seqNo)
{
  uint64_t tableIndex = (seqNo >> client->sampleFreq) & client->sampleIndexMask;
  assert((tableIndex >> client->sampleTableSize) == 0);
  return (NdnpingClientSample*)client->sampleTable + tableIndex;
}

static void
NdnpingClient_PrepareTxInterest(NdnpingClient* client, struct rte_mbuf* pkt)
{
  uint64_t seqNo = ++client->suffixComponent.compV;
  int patternId = NdnpingClient_SelectPattern(client, seqNo);
  NdnpingClientPattern* pattern =
    NameSet_GetUsrT(&client->patterns, patternId, NdnpingClientPattern*);
  ++pattern->nInterests;

  client->interestTpl.namePrefix = NameSet_GetName(
    &client->patterns, patternId, &client->interestTpl.namePrefixSize);
  EncodeInterest(pkt, &client->interestTpl);
  Packet_SetNdnPktType(pkt, NdnPktType_Interest);
  ZF_LOGV("%" PRI_FaceId " <I seq=%" PRIx64 " pattern=%d", client->face->id,
          seqNo, patternId);

  if (!NdnpingClient_IsSamplingEnabled(client)) {
    return;
  }
  if (patternId != PATTERN_0) {
    return;
  }
  if (likely((seqNo & client->samplingMask) != 0)) {
    return;
  }

  NdnpingClientSample* sample = NdnpingClient_FindSample(client, seqNo);
  if (unlikely(sample->isPending)) { // timeout
    assert(sample->patternId == PATTERN_0);
  }
  sample->isPending = true;
  sample->patternId = patternId;
  sample->sendTime = rte_get_tsc_cycles() >> NDNPING_TIMING_PRECISION;
}

static void
NdnpingClient_TxBurst(NdnpingClient* client)
{
  struct rte_mbuf* pkts[NDNPINGCLIENT_TX_BURST_SIZE];
  int res = rte_pktmbuf_alloc_bulk(client->mpInterest, pkts,
                                   NDNPINGCLIENT_TX_BURST_SIZE);
  if (unlikely(res != 0)) {
    ZF_LOGW("%" PRI_FaceId " TX alloc failure %d", client->face->id, res);
    return;
  }

  for (uint16_t i = 0; i < NDNPINGCLIENT_TX_BURST_SIZE; ++i) {
    NdnpingClient_PrepareTxInterest(client, pkts[i]);
  }
  Face_TxBurst(client->face, pkts, NDNPINGCLIENT_TX_BURST_SIZE);
  FreeMbufs(pkts, NDNPINGCLIENT_TX_BURST_SIZE);
}

static bool
NdnpingClient_GetSeqNoFromName(const Name* name, uint64_t* seqNo)
{
  if (unlikely(name->nComps < 1)) {
    return false;
  }
  TlvElement comp;
  Name_GetComp(name, name->nComps - 1, &comp);
  if (unlikely(comp.length != 8)) {
    return false;
  }
  return MbufLoc_ReadU64(&comp.value, seqNo);
}

static void
NdnpingClient_ProcessRxData(NdnpingClient* client, struct rte_mbuf* pkt)
{
  const DataPkt* data = Packet_GetDataHdr(pkt);
  uint64_t seqNo;
  if (!unlikely(NdnpingClient_GetSeqNoFromName(&data->name, &seqNo))) {
    return;
  }

  int patternId = NdnpingClient_SelectPattern(client, seqNo);
  ZF_LOGV("%" PRI_FaceId " >D seq=%" PRIx64 " pattern=%d", client->face->id,
          seqNo, patternId);

  NdnpingClientPattern* pattern =
    NameSet_GetUsrT(&client->patterns, patternId, NdnpingClientPattern*);
  ++pattern->nData;

  if (!NdnpingClient_IsSamplingEnabled(client)) {
    return;
  }
  if (patternId != PATTERN_0) {
    return;
  }
  if (likely((seqNo & client->samplingMask) != 0)) {
    return;
  }
  NdnpingClientSample* sample = NdnpingClient_FindSample(client, seqNo);
  assert(sample->patternId == PATTERN_0);
  if (unlikely(!sample->isPending)) {
    ZF_LOGI("%" PRI_FaceId " duplicate-Data-or-Nack seq=%" PRIx64,
            client->face->id, seqNo);
    return;
  }
  sample->isPending = false;

  uint64_t now = rte_get_tsc_cycles() >> NDNPING_TIMING_PRECISION;
  RunningStat_Push(&pattern->rtt, now - sample->sendTime);
}

static void
NdnpingClient_ProcessRxNack(NdnpingClient* client, struct rte_mbuf* pkt)
{
  const LpPkt* lpp = Packet_GetLpHdr(pkt);
  const InterestPkt* interest = Packet_GetInterestHdr(pkt);
  uint64_t seqNo;
  if (!unlikely(NdnpingClient_GetSeqNoFromName(&interest->name, &seqNo))) {
    return;
  }

  int patternId = NdnpingClient_SelectPattern(client, seqNo);
  ZF_LOGV("%" PRI_FaceId " >N seq=%" PRIx64 " pattern=%d", client->face->id,
          seqNo, patternId);

  NdnpingClientPattern* pattern =
    NameSet_GetUsrT(&client->patterns, patternId, NdnpingClientPattern*);
  ++pattern->nNacks;

  if (!NdnpingClient_IsSamplingEnabled(client)) {
    return;
  }
  if (patternId != PATTERN_0) {
    return;
  }
  if (likely((seqNo & client->samplingMask) != 0)) {
    return;
  }
  NdnpingClientSample* sample = NdnpingClient_FindSample(client, seqNo);
  assert(sample->patternId == PATTERN_0);
  if (unlikely(!sample->isPending)) {
    ZF_LOGI("%" PRI_FaceId " duplicate-Data-or-Nack seq=%" PRIx64,
            client->face->id, seqNo);
    return;
  }
  sample->isPending = false;
}

static void
NdnpingClient_RxBurst(NdnpingClient* client)
{
  struct rte_mbuf* pkts[NDNPINGCLIENT_RX_BURST_SIZE];
  uint16_t nRx = Face_RxBurst(client->face, pkts, NDNPINGCLIENT_RX_BURST_SIZE);
  for (uint16_t i = 0; i < nRx; ++i) {
    struct rte_mbuf* pkt = pkts[i];
    NdnPktType l3type = Packet_GetNdnPktType(pkt);
    if (likely(l3type == NdnPktType_Data)) {
      NdnpingClient_ProcessRxData(client, pkt);
    } else if (likely(l3type == NdnPktType_Nack)) {
      NdnpingClient_ProcessRxNack(client, pkt);
    }
  }
  FreeMbufs(pkts, nRx);
}

void
NdnpingClient_Run(NdnpingClient* client)
{
  uint64_t tscHz = rte_get_tsc_hz();
  atomic_store_explicit(&client->tscHz, tscHz, memory_order_relaxed);
  uint64_t txBurstInterval =
    client->interestInterval / 1000.0 * tscHz * NDNPINGCLIENT_TX_BURST_SIZE;
  ZF_LOGI("%" PRI_FaceId " starting %p tx-burst-interval=%" PRIu64 " @%" PRIu64
          "Hz",
          client->face->id, client, txBurstInterval, tscHz);

  uint64_t nextTxBurst = rte_get_tsc_cycles();
  while (true) {
    uint64_t now = rte_get_tsc_cycles();
    if (now > nextTxBurst) {
      NdnpingClient_TxBurst(client);
      nextTxBurst += txBurstInterval;
    }
    NdnpingClient_RxBurst(client);
  }
}

uint64_t
NdnpingClient_GetTscHz(NdnpingClient* client)
{
  return atomic_load_explicit(&client->tscHz, memory_order_relaxed);
}
