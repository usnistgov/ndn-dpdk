#include "server.h"

#include "../core/logger.h"

INIT_ZF_LOG(PingServer);

__attribute__((nonnull)) static int
PingServer_FindPattern(PingServer* server, LName name)
{
  for (uint16_t i = 0; i < server->nPatterns; ++i) {
    PingServerPattern* pattern = &server->pattern[i];
    if (pattern->prefix.length <= name.length &&
        memcmp(pattern->prefix.value, name.value, pattern->prefix.length) == 0) {
      return i;
    }
  }
  return -1;
}

__attribute__((nonnull)) static PingReplyId
PingServer_SelectReply(PingServer* server, PingServerPattern* pattern)
{
  uint32_t rnd = pcg32_random_r(&server->replyRng);
  return pattern->weight[rnd % pattern->nWeights];
}

__attribute__((nonnull)) static Packet*
PingServer_RespondData(PingServer* server, PingServerPattern* pattern, PingServerReply* reply,
                       Packet* npkt)
{
  LpL3* lpl3 = Packet_GetLpL3Hdr(npkt);
  uint64_t token = lpl3->pitToken;
  const LName* name = (const LName*)&Packet_GetInterestHdr(npkt)->name;

  struct rte_mbuf* segs[2];

  segs[0] = rte_pktmbuf_alloc(server->dataMp);
  if (unlikely(segs[0] == NULL)) {
    ZF_LOGW("dataMp-full");
    ++server->nAllocError;
    rte_pktmbuf_free(Packet_ToMbuf(npkt));
    return NULL;
  }

  segs[1] = rte_pktmbuf_alloc(server->indirectMp);
  if (unlikely(segs[1] == NULL)) {
    ZF_LOGW("indirectMp-full");
    ++server->nAllocError;
    segs[1] = Packet_ToMbuf(npkt);
    rte_pktmbuf_free_bulk_(segs, 2);
    return NULL;
  }

  Packet* output = DataGen_Encode(reply->dataGen, segs[0], segs[1], *name);
  rte_pktmbuf_free(Packet_ToMbuf(npkt));
  Packet_GetLpL3Hdr(output)->pitToken = token;
  return output;
}

__attribute__((nonnull)) static Packet*
PingServer_RespondNack(PingServer* server, PingServerPattern* pattern, PingServerReply* reply,
                       Packet* npkt)
{
  return Nack_FromInterest(npkt, reply->nackReason);
}

__attribute__((nonnull)) static Packet*
PingServer_RespondTimeout(PingServer* server, PingServerPattern* pattern, PingServerReply* reply,
                          Packet* npkt)
{
  rte_pktmbuf_free(Packet_ToMbuf(npkt));
  return NULL;
}

typedef Packet* (*PingServer_Respond)(PingServer* server, PingServerPattern* pattern,
                                      PingServerReply* reply, Packet* npkt);

static const PingServer_Respond PingServer_RespondJmp[3] = {
  [PINGSERVER_REPLY_DATA] = PingServer_RespondData,
  [PINGSERVER_REPLY_NACK] = PingServer_RespondNack,
  [PINGSERVER_REPLY_TIMEOUT] = PingServer_RespondTimeout,
};

__attribute__((nonnull)) static Packet*
PingServer_ProcessInterest(PingServer* server, Packet* npkt)
{
  uint64_t token = Packet_GetLpL3Hdr(npkt)->pitToken;
  const LName* name = (const LName*)&Packet_GetInterestHdr(npkt)->name;

  int patternId = PingServer_FindPattern(server, *name);
  if (unlikely(patternId < 0)) {
    ZF_LOGD(">I dn-token=%016" PRIx64 " no-pattern", token);
    ++server->nNoMatch;
    if (server->wantNackNoRoute) {
      return Nack_FromInterest(npkt, NackNoRoute);
    } else {
      rte_pktmbuf_free(Packet_ToMbuf(npkt));
      return NULL;
    }
  }

  PingServerPattern* pattern = &server->pattern[patternId];
  uint8_t replyId = PingServer_SelectReply(server, pattern);
  PingServerReply* reply = &pattern->reply[replyId];

  ZF_LOGD(">I dn-token=%016" PRIx64 " pattern=%d reply=%" PRIu8, token, patternId, replyId);
  ++reply->nInterests;
  return PingServer_RespondJmp[reply->kind](server, pattern, reply, npkt);
}

int
PingServer_Run(PingServer* server)
{
  struct rte_mbuf* rx[MaxBurstSize];
  Packet* tx[MaxBurstSize];

  while (ThreadStopFlag_ShouldContinue(&server->stop)) {
    TscTime now = rte_get_tsc_cycles();
    PktQueuePopResult pop = PktQueue_Pop(&server->rxQueue, rx, MaxBurstSize, now);
    if (unlikely(pop.count == 0)) {
      rte_pause();
      continue;
    }

    uint16_t nTx = 0;
    for (uint16_t i = 0; i < pop.count; ++i) {
      Packet* npkt = Packet_FromMbuf(rx[i]);
      assert(Packet_GetType(npkt) == PktInterest);
      tx[nTx] = PingServer_ProcessInterest(server, npkt);
      nTx += (int)(tx[nTx] != NULL);
    }

    ZF_LOGD("face=%" PRI_FaceID "nRx=%" PRIu16 " nTx=%" PRIu16, server->face, pop.count, nTx);
    Face_TxBurst(server->face, tx, nTx);
  }
  return 0;
}
