#include "server.h"

#include "../../core/logger.h"

INIT_ZF_LOG(PingServer);

static int
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

static PingReplyId
PingServer_SelectReply(PingServer* server, PingServerPattern* pattern)
{
  uint32_t rnd = pcg32_random_r(&server->replyRng);
  return pattern->weight[rnd % pattern->nWeights];
}

static Packet*
PingServer_RespondData(PingServer* server,
                       PingServerPattern* pattern,
                       PingServerReply* reply,
                       Packet* npkt)
{
  uint64_t token = Packet_GetLpL3Hdr(npkt)->pitToken;
  const LName name = *(const LName*)&Packet_GetInterestHdr(npkt)->name;

  struct rte_mbuf* seg0 = rte_pktmbuf_alloc(server->dataMp);
  if (unlikely(seg0 == NULL)) {
    ZF_LOGW("dataMp-full");
    ++server->nAllocError;
    rte_pktmbuf_free(Packet_ToMbuf(npkt));
    return NULL;
  }
  struct rte_mbuf* seg1 = rte_pktmbuf_alloc(server->indirectMp);
  if (unlikely(seg0 == NULL)) {
    ZF_LOGW("indirectMp-full");
    ++server->nAllocError;
    rte_pktmbuf_free(Packet_ToMbuf(npkt));
    rte_pktmbuf_free(seg0);
    return NULL;
  }

  DataGen_Encode(reply->dataGen, seg0, seg1, name);
  rte_pktmbuf_free(Packet_ToMbuf(npkt));

  Packet* response = Packet_FromMbuf(seg0);
  Packet_SetL2PktType(response, L2PktType_None);
  Packet_InitLpL3Hdr(response)->pitToken = token;
  Packet_SetL3PktType(response, L3PktType_Data); // for stats; no PData*
  return response;
}

static Packet*
PingServer_RespondNack(PingServer* server,
                       PingServerPattern* pattern,
                       PingServerReply* reply,
                       Packet* npkt)
{
  MakeNack(npkt, reply->nackReason);
  return npkt;
}

static Packet*
PingServer_RespondTimeout(PingServer* server,
                          PingServerPattern* pattern,
                          PingServerReply* reply,
                          Packet* npkt)
{
  rte_pktmbuf_free(Packet_ToMbuf(npkt));
  return NULL;
}

typedef Packet* (*PingServer_Respond)(PingServer* server,
                                      PingServerPattern* pattern,
                                      PingServerReply* reply,
                                      Packet* npkt);

static const PingServer_Respond PingServer_RespondJmp[3] = {
  [PINGSERVER_REPLY_DATA] = PingServer_RespondData,
  [PINGSERVER_REPLY_NACK] = PingServer_RespondNack,
  [PINGSERVER_REPLY_TIMEOUT] = PingServer_RespondTimeout,
};

static Packet*
PingServer_ProcessInterest(PingServer* server, Packet* npkt)
{
  uint64_t token = Packet_GetLpL3Hdr(npkt)->pitToken;
  const LName name = *(const LName*)&Packet_GetInterestHdr(npkt)->name;

  int patternId = PingServer_FindPattern(server, name);
  if (unlikely(patternId < 0)) {
    ZF_LOGD(">I dn-token=%016" PRIx64 " no-pattern", token);
    ++server->nNoMatch;
    if (server->wantNackNoRoute) {
      MakeNack(npkt, NackReason_NoRoute);
      return npkt;
    } else {
      rte_pktmbuf_free(Packet_ToMbuf(npkt));
      return NULL;
    }
  }

  PingServerPattern* pattern = &server->pattern[patternId];
  uint8_t replyId = PingServer_SelectReply(server, pattern);
  PingServerReply* reply = &pattern->reply[replyId];

  ZF_LOGD(">I dn-token=%016" PRIx64 " pattern=%d reply=%" PRIu8,
          token,
          patternId,
          replyId);
  ++reply->nInterests;
  return PingServer_RespondJmp[reply->kind](server, pattern, reply, npkt);
}

void
PingServer_Run(PingServer* server)
{
  Packet* rx[PINGSERVER_BURST_SIZE];
  Packet* tx[PINGSERVER_BURST_SIZE];

  while (ThreadStopFlag_ShouldContinue(&server->stop)) {
    uint16_t nRx = rte_ring_sc_dequeue_burst(
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
