#include "server.h"

#include "../../core/logger.h"
#include "../../ndn/encode-data.h"

INIT_ZF_LOG(PingServer);

static uint8_t PingServer_payloadV[PINGSERVER_PAYLOAD_MAX];

static uint16_t
PingServer_FindPattern(PingServer* server, LName name)
{
  for (uint16_t i = 0; i < server->nPatterns; ++i) {
    PingServerPattern* pattern = &server->pattern[i];
    if (pattern->prefix.length <= name.length &&
        memcmp(pattern->prefix.value, name.value, pattern->prefix.length) ==
          0) {
      return i;
    }
  }
  return -1;
}

static Packet*
PingServer_MakeData(PingServer* server, PingServerPattern* pattern, LName name)
{
  struct rte_mbuf* m = rte_pktmbuf_alloc(server->dataMp);
  if (unlikely(m == NULL)) {
    ZF_LOGW("dataMp-full");
    return NULL;
  }
  m->data_off = server->dataMbufHeadroom;
  EncodeData(m,
             name,
             pattern->suffix,
             pattern->freshnessPeriod,
             pattern->payloadL,
             PingServer_payloadV);

  Packet* npkt = Packet_FromMbuf(m);
  Packet_SetL2PktType(npkt, L2PktType_None);
  Packet_SetL3PktType(npkt, L3PktType_Data); // for stats; no PData*
  return npkt;
}

static Packet*
PingServer_ProcessInterest(PingServer* server, Packet* npkt)
{
  struct rte_mbuf* pkt = Packet_ToMbuf(npkt);
  uint64_t token = Packet_GetLpL3Hdr(npkt)->pitToken;

  const LName name = *(const LName*)&Packet_GetInterestHdr(npkt)->name;
  uint16_t patternId = PingServer_FindPattern(server, name);
  if (unlikely(patternId < 0)) {
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

  ZF_LOGD(">I dn-token=%016" PRIx64 " pattern=%" PRIu16, token, patternId);
  PingServerPattern* pattern = &server->pattern[patternId];
  ++pattern->nInterests;

  Packet* dataPkt = PingServer_MakeData(server, pattern, name);
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
PingServer_Run(PingServer* server)
{
  Packet* rx[PINGSERVER_BURST_SIZE];
  Packet* tx[PINGSERVER_BURST_SIZE];

  while (ThreadStopFlag_ShouldContinue(&server->stop)) {
    uint16_t nRx = rte_ring_sc_dequeue_bulk(
      server->rxQueue, (void**)rx, PINGSERVER_BURST_SIZE, NULL);
    uint16_t nTx = 0;
    for (uint16_t i = 0; i < nRx; ++i) {
      Packet* npkt = rx[i];
      assert(Packet_GetL3PktType(npkt) == L3PktType_Interest);
      tx[nTx] = PingServer_ProcessInterest(server, npkt);
      nTx += (tx[nTx] != NULL);
    }
    if (likely(nRx > 0)) {
      ZF_LOGD("face=%" PRI_FaceId "nRx=%" PRIu16 " nTx=%" PRIu16,
              server->face,
              nRx,
              nTx);
    }
    Face_TxBurst(server->face, tx, nTx);
  }
}
