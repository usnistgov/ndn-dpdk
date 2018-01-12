#include "client.h"

#include "../../core/logger.h"

#include <rte_cycles.h>
#include <rte_random.h>

INIT_ZF_LOG(NdnpingClient);

#define NDNPINGCLIENT_RX_BURST_SIZE 64
#define NDNPINGCLIENT_TX_BURST_SIZE 64
#define NDNPINGCLIENT_INTEREST_LIFETIME 1000

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
}

static inline void
NdnpingClient_PrepareTxInterest(NdnpingClient* client, struct rte_mbuf* pkt)
{
  uint64_t seqNo = ++client->suffixComponent.compV;
  int patternId = seqNo % client->prefixes.nRecords;
  client->interestTpl.namePrefix = NameSet_GetName(
    &client->prefixes, patternId, &client->interestTpl.namePrefixSize);
  EncodeInterest(pkt, &client->interestTpl);
  ZF_LOGV("%" PRI_FaceId " <I seq=%" PRIx64 " pattern=%d", client->face->id,
          seqNo, patternId);
}

static inline void
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

static inline bool
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

static inline void
NdnpingClient_ProcessRxData(NdnpingClient* client, struct rte_mbuf* pkt)
{
  const DataPkt* data = Packet_GetDataHdr(pkt);
  uint64_t seqNo;
  if (!unlikely(NdnpingClient_GetSeqNoFromName(&data->name, &seqNo))) {
    return;
  }

  ZF_LOGV("%" PRI_FaceId " >D seq=%" PRIx64, client->face->id, seqNo);
}

static inline void
NdnpingClient_ProcessRxNack(NdnpingClient* client, struct rte_mbuf* pkt)
{
  const LpPkt* lpp = Packet_GetLpHdr(pkt);
  const InterestPkt* interest = Packet_GetInterestHdr(pkt);
  uint64_t seqNo;
  if (!unlikely(NdnpingClient_GetSeqNoFromName(&interest->name, &seqNo))) {
    return;
  }

  ZF_LOGV("%" PRI_FaceId " >N seq=%" PRIx64 " pattern=%d", client->face->id,
          seqNo);
}

static inline void
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

int
NdnpingClient_Run(NdnpingClient* client)
{
  uint64_t txBurstInterval = client->interestInterval / 1000.0 *
                             rte_get_tsc_hz() * NDNPINGCLIENT_TX_BURST_SIZE;
  ZF_LOGI("%" PRI_FaceId " starting %p burst-interval=%" PRIu64 " @%" PRIu64
          "Hz",
          client->face->id, client, txBurstInterval, rte_get_tsc_hz());
  assert(txBurstInterval > 0);

  uint64_t nextTxBurst = rte_get_tsc_cycles();
  while (true) {
    uint64_t now = rte_get_tsc_cycles();
    if (now > nextTxBurst) {
      NdnpingClient_TxBurst(client);
      nextTxBurst += txBurstInterval;
    }
    NdnpingClient_RxBurst(client);
  }
  return 0;
}
