#include "producer.h"

#include "../core/logger.h"

N_LOG_INIT(Tgp);

__attribute__((nonnull)) static int
Tgp_FindPattern(Tgp* p, LName name)
{
  for (uint16_t i = 0; i < p->nPatterns; ++i) {
    TgpPattern* pattern = &p->pattern[i];
    if (pattern->prefix.length <= name.length &&
        memcmp(pattern->prefix.value, name.value, pattern->prefix.length) == 0) {
      return i;
    }
  }
  return -1;
}

__attribute__((nonnull)) static TgpReplyID
Tgp_SelectReply(Tgp* p, TgpPattern* pattern)
{
  uint32_t w = pcg32_boundedrand_r(&p->replyRng, pattern->nWeights);
  return pattern->weight[w];
}

__attribute__((nonnull)) static Packet*
Tgp_RespondData(Tgp* p, TgpPattern* pattern, TgpReply* reply, Packet* npkt)
{
  const LName* name = (const LName*)&Packet_GetInterestHdr(npkt)->name;
  Packet* output = DataGen_Encode(&reply->dataGen, *name, &p->mp, Face_PacketTxAlign(p->face));
  if (likely(output != NULL)) {
    Packet_GetLpL3Hdr(output)->pitToken = Packet_GetLpL3Hdr(npkt)->pitToken;
  }
  rte_pktmbuf_free(Packet_ToMbuf(npkt));
  return output;
}

__attribute__((nonnull)) static Packet*
Tgp_RespondNack(Tgp* p, TgpPattern* pattern, TgpReply* reply, Packet* npkt)
{
  return Nack_FromInterest(npkt, reply->nackReason, &p->mp, Face_PacketTxAlign(p->face));
}

__attribute__((nonnull)) static Packet*
Tgp_RespondTimeout(Tgp* p, TgpPattern* pattern, TgpReply* reply, Packet* npkt)
{
  rte_pktmbuf_free(Packet_ToMbuf(npkt));
  return NULL;
}

typedef Packet* (*Tgp_Respond)(Tgp* p, TgpPattern* pattern, TgpReply* reply, Packet* npkt);

static const Tgp_Respond Tgp_RespondJmp[3] = {
  [TgpReplyData] = Tgp_RespondData,
  [TgpReplyNack] = Tgp_RespondNack,
  [TgpReplyTimeout] = Tgp_RespondTimeout,
};

__attribute__((nonnull)) static Packet*
Tgp_ProcessInterest(Tgp* p, Packet* npkt)
{
  uint64_t token = Packet_GetLpL3Hdr(npkt)->pitToken;
  const LName* name = (const LName*)&Packet_GetInterestHdr(npkt)->name;

  int patternId = Tgp_FindPattern(p, *name);
  if (unlikely(patternId < 0)) {
    N_LOGD(">I dn-token=%016" PRIx64 " no-pattern", token);
    ++p->nNoMatch;
    rte_pktmbuf_free(Packet_ToMbuf(npkt));
    return NULL;
  }

  TgpPattern* pattern = &p->pattern[patternId];
  uint8_t replyId = Tgp_SelectReply(p, pattern);
  TgpReply* reply = &pattern->reply[replyId];

  N_LOGD(">I dn-token=%016" PRIx64 " pattern=%d reply=%" PRIu8, token, patternId, replyId);
  ++reply->nInterests;
  return Tgp_RespondJmp[reply->kind](p, pattern, reply, npkt);
}

int
Tgp_Run(Tgp* p)
{
  struct rte_mbuf* rx[MaxBurstSize];
  Packet* tx[MaxBurstSize];

  while (ThreadStopFlag_ShouldContinue(&p->stop)) {
    TscTime now = rte_get_tsc_cycles();
    PktQueuePopResult pop = PktQueue_Pop(&p->rxQueue, rx, MaxBurstSize, now);
    if (unlikely(pop.count == 0)) {
      rte_pause();
      continue;
    }

    uint16_t nTx = 0;
    for (uint16_t i = 0; i < pop.count; ++i) {
      Packet* npkt = Packet_FromMbuf(rx[i]);
      NDNDPDK_ASSERT(Packet_GetType(npkt) == PktInterest);
      tx[nTx] = Tgp_ProcessInterest(p, npkt);
      nTx += (int)(tx[nTx] != NULL);
    }

    N_LOGD("burst face=%" PRI_FaceID "nRx=%" PRIu16 " nTx=%" PRIu16, p->face, pop.count, nTx);
    Face_TxBurst(p->face, tx, nTx);
  }
  return 0;
}
