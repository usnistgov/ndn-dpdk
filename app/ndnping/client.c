#include "client.h"

#include "../../core/logger.h"

#include <rte_random.h>

INIT_ZF_LOG(NdnpingClient);

#define NDNPINGCLIENT_TX_BURST_SIZE 64
#define NDNPINGCLIENT_INTEREST_LIFETIME 1000

typedef struct NdnpingClientSample
{
  bool isPending : 1;
  bool _reserved : 1;
  int patternId : 6;
  uint64_t sendTime : 56; ///< TscTime >> NDNPING_TIMING_PRECISION
} __rte_packed NdnpingClientSample;
static_assert(sizeof(NdnpingClientSample) == sizeof(uint64_t), "");

void
NdnpingClient_Init(NdnpingClient* client)
{
  static_assert(sizeof(client->suffixComponent) == 16, "");
  client->suffixComponent.compT = TT_GenericNameComponent;
  client->suffixComponent.compL = 8;
  client->suffixComponent.compV = rte_rand();

  client->interestTpl.canBePrefix = false;
  client->interestTpl.mustBeFresh = true;
  client->interestTpl.fhL = 0;
  client->interestTpl.fhV = NULL;
  client->interestTpl.lifetime = NDNPINGCLIENT_INTEREST_LIFETIME;
  client->interestTpl.hopLimit = 255;

  uint16_t res = InterestTemplate_Prepare(
    &client->interestTpl, client->interestPrepareBuffer,
    sizeof(client->interestPrepareBuffer));
  assert(res == 0);
  NonceGen_Init(&client->nonceGen);

  client->sampleTable = NULL;
}

void
NdnpingClient_EnableSampling(NdnpingClient* client, int numaSocket)
{
  size_t nSampleTableEntries = 1 << client->sampleTableSize;
  client->samplingMask = (1 << client->sampleFreq) - 1;
  client->sampleIndexMask = (1 << client->sampleTableSize) - 1;
  client->sampleTable =
    rte_calloc_socket("NdnpingClient.sampleTable", nSampleTableEntries,
                      sizeof(NdnpingClientSample), 0, numaSocket);
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
  if (client->sampleTable == NULL) { // sampling disabled
    return NULL;
  }
  if (likely((seqNo & client->samplingMask) != 0)) { // seqNo not sampled
    return NULL;
  }

  uint64_t tableIndex = (seqNo >> client->sampleFreq) & client->sampleIndexMask;
  assert((tableIndex >> client->sampleTableSize) == 0);
  return (NdnpingClientSample*)client->sampleTable + tableIndex;
}

static void
NdnpingClient_PrepareTxInterest(NdnpingClient* client, Packet* npkt)
{
  struct rte_mbuf* pkt = Packet_ToMbuf(npkt);
  pkt->data_off = client->interestMbufHeadroom;
  uint64_t seqNo = ++client->suffixComponent.compV;
  int patternId = NdnpingClient_SelectPattern(client, seqNo);
  NdnpingClientPattern* pattern =
    NameSet_GetUsrT(&client->patterns, patternId, NdnpingClientPattern*);
  ++pattern->nInterests;

  client->interestTpl.namePrefix =
    NameSet_GetName(&client->patterns, patternId);
  LName nameSuffix = {.length = 10, .value = &client->suffixComponent.compT };
  EncodeInterest(pkt, &client->interestTpl, client->interestPrepareBuffer,
                 nameSuffix, NonceGen_Next(&client->nonceGen), 0, NULL);
  Packet_SetL3PktType(npkt, L3PktType_Interest); // for stats; no PInterest*
  ZF_LOGD("<I seq=%" PRIx64 " pattern=%d", seqNo, patternId);

  NdnpingClientSample* sample = NdnpingClient_FindSample(client, seqNo);
  if (sample == NULL) {
    return;
  }
  if (unlikely(sample->isPending)) { // timeout
    ZF_LOGD("TIMEOUT pattern=%d", sample->patternId);
  }
  sample->isPending = true;
  sample->patternId = patternId;
  sample->sendTime = rte_get_tsc_cycles() >> NDNPING_TIMING_PRECISION;
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

  for (uint16_t i = 0; i < NDNPINGCLIENT_TX_BURST_SIZE; ++i) {
    NdnpingClient_PrepareTxInterest(client, npkts[i]);
  }
  Face_TxBurst(client->face, npkts, NDNPINGCLIENT_TX_BURST_SIZE);
}

void
NdnpingClient_RunTx(NdnpingClient* client)
{
  uint64_t tscHz = rte_get_tsc_hz();
  uint64_t txBurstInterval =
    client->interestInterval / 1000.0 * tscHz * NDNPINGCLIENT_TX_BURST_SIZE;
  ZF_LOGI("face=%" PRI_FaceId " client=%p "
          "tx-burst-interval=%" PRIu64 " @%" PRIu64 "Hz",
          client->face, client, txBurstInterval, tscHz);

  uint64_t nextTxBurst = rte_get_tsc_cycles();
  while (true) {
    if (rte_get_tsc_cycles() < nextTxBurst) {
      rte_pause();
      continue;
    }
    NdnpingClient_TxBurst(client);
    nextTxBurst += txBurstInterval;
  }
}

static bool
NdnpingClient_GetSeqNoFromName(const Name* name, uint64_t* seqNo)
{
  if (unlikely(name->p.nComps < 1)) {
    return false;
  }

  NameComp comp = Name_GetComp(name, name->p.nComps - 1);
  if (unlikely(comp.size != 10)) {
    return false;
  }

  *seqNo = *(const unaligned_uint64_t*)RTE_PTR_ADD(comp.tlv, 2);
  return true;
}

static void
NdnpingClient_SampleDataOrNack(NdnpingClient* client, uint64_t seqNo,
                               int patternId, NdnpingClientPattern* pattern,
                               bool isData)
{
  NdnpingClientSample* sample = NdnpingClient_FindSample(client, seqNo);
  if (sample == NULL) {
    return;
  }
  if (unlikely(sample->patternId != patternId)) {
    ZF_LOGD("^ mismatch-sample-pattern=%d", sample->patternId);
    return;
  }
  if (unlikely(!sample->isPending)) {
    ZF_LOGD("^ duplicate-Data-or-Nack");
    return;
  }
  sample->isPending = false;

  if (isData) {
    uint64_t now = rte_get_tsc_cycles() >> NDNPING_TIMING_PRECISION;
    RunningStat_Push(&pattern->rtt, now - sample->sendTime);
  }
}

static void
NdnpingClient_ProcessRxData(NdnpingClient* client, Packet* npkt)
{
  const PData* data = Packet_GetDataHdr(npkt);
  uint64_t seqNo;
  if (!unlikely(NdnpingClient_GetSeqNoFromName(&data->name, &seqNo))) {
    return;
  }

  int patternId = NdnpingClient_SelectPattern(client, seqNo);
  ZF_LOGD(">D seq=%" PRIx64 " pattern=%d", seqNo, patternId);

  NdnpingClientPattern* pattern =
    NameSet_GetUsrT(&client->patterns, patternId, NdnpingClientPattern*);
  ++pattern->nData;

  NdnpingClient_SampleDataOrNack(client, seqNo, patternId, pattern, true);
}

static void
NdnpingClient_ProcessRxNack(NdnpingClient* client, Packet* npkt)
{
  const PNack* nack = Packet_GetNackHdr(npkt);
  uint64_t seqNo;
  if (!unlikely(NdnpingClient_GetSeqNoFromName(&nack->interest.name, &seqNo))) {
    return;
  }

  int patternId = NdnpingClient_SelectPattern(client, seqNo);
  ZF_LOGD(">N seq=%" PRIx64 " pattern=%d", seqNo, patternId);

  NdnpingClientPattern* pattern =
    NameSet_GetUsrT(&client->patterns, patternId, NdnpingClientPattern*);
  ++pattern->nNacks;

  NdnpingClient_SampleDataOrNack(client, seqNo, patternId, pattern, false);
}

void
NdnpingClient_Rx(FaceRxBurst* burst, void* client0)
{
  NdnpingClient* client = (NdnpingClient*)client0;
  for (uint16_t i = 0; i < burst->nData; ++i) {
    NdnpingClient_ProcessRxData(client, FaceRxBurst_GetData(burst, i));
  }
  for (uint16_t i = 0; i < burst->nNacks; ++i) {
    NdnpingClient_ProcessRxNack(client, FaceRxBurst_GetNack(burst, i));
  }

  FreeMbufs((struct rte_mbuf**)FaceRxBurst_ListInterests(burst),
            burst->nInterests);
  FreeMbufs((struct rte_mbuf**)FaceRxBurst_ListData(burst), burst->nData);
  FreeMbufs((struct rte_mbuf**)FaceRxBurst_ListNacks(burst), burst->nNacks);
}
