#include "server.h"

#include "../../core/logger.h"
#include "../../ndn/encode-data.h"

INIT_ZF_LOG(NdnpingServer);

#define NDNPINGSERVER_BURST_SIZE 64

static uint8_t NdnpingServer_payloadV[NDNPINGSERVER_PAYLOAD_MAX];

static Packet*
NdnpingServer_MakeData(NdnpingServer* server, NdnpingServerPattern* pattern,
                       LName name)
{
  struct rte_mbuf* m = rte_pktmbuf_alloc(server->dataMp);
  if (unlikely(m == NULL)) {
    ZF_LOGW("dataMp-full");
    return NULL;
  }
  m->data_off = server->dataMbufHeadroom;
  EncodeData(m, name, pattern->nameSuffix, server->freshnessPeriod,
             pattern->payloadL, NdnpingServer_payloadV);

  Packet* npkt = Packet_FromMbuf(m);
  Packet_SetL2PktType(npkt, L2PktType_None);
  Packet_SetL3PktType(npkt, L3PktType_Data); // for stats; no PData*
  return npkt;
}

static Packet*
NdnpingServer_ProcessInterest(NdnpingServer* server, Packet* npkt)
{
  struct rte_mbuf* pkt = Packet_ToMbuf(npkt);
  uint64_t token = Packet_GetLpL3Hdr(npkt)->pitToken;

  const LName name = *(const LName*)&Packet_GetInterestHdr(npkt)->name;

  int patternId = NameSet_FindPrefix(&server->patterns, name);
  if (patternId < 0) {
    ZF_LOGD(">I dn-token=%016" PRIx64 " no-pattern", token);
    ++server->nNoMatch;
    if (server->wantNackNoRoute) {
      MakeNack(npkt, NackReason_NoRoute);
      return npkt;
    } else {
      rte_pktmbuf_free(pkt);
      return NULL;
    }
  }
  ZF_LOGD(">I dn-token=%016" PRIx64 " pattern=%d", token, patternId);

  NdnpingServerPattern* pattern =
    NameSet_GetUsrT(&server->patterns, patternId, NdnpingServerPattern*);
  ++pattern->nInterests;

  Packet* dataPkt = NdnpingServer_MakeData(server, pattern, name);
  if (unlikely(dataPkt == NULL)) {
    ++server->nAllocError;
    MakeNack(npkt, NackReason_Congestion);
    return npkt;
  }

  Packet_InitLpL3Hdr(dataPkt)->pitToken = token;
  rte_pktmbuf_free(pkt);
  return dataPkt;
}

void
NdnpingServer_Run(NdnpingServer* server)
{
  const int burstSize = 64;
  Packet* rx[burstSize];
  Packet* tx[burstSize];

  while (true) {
    uint16_t nRx =
      rte_ring_sc_dequeue_bulk(server->rxQueue, (void**)rx, burstSize, NULL);
    uint16_t nTx = 0;
    for (uint16_t i = 0; i < nRx; ++i) {
      Packet* npkt = rx[i];
      assert(Packet_GetL3PktType(npkt) == L3PktType_Interest);
      tx[nTx] = NdnpingServer_ProcessInterest(server, npkt);
      nTx += (tx[nTx] != NULL);
    }
    if (likely(nRx > 0)) {
      ZF_LOGD("face=%" PRI_FaceId "nRx=%" PRIu16 " nTx=%" PRIu16, server->face,
              nRx, nTx);
    }
    Face_TxBurst(server->face, tx, nTx);
  }
}
