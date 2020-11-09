#include "producer.h"

#include "../core/logger.h"

INIT_ZF_LOG(TgProducer);

__attribute__((nonnull)) static int
TgProducer_FindPattern(TgProducer* producer, LName name)
{
  for (uint16_t i = 0; i < producer->nPatterns; ++i) {
    TgProducerPattern* pattern = &producer->pattern[i];
    if (pattern->prefix.length <= name.length &&
        memcmp(pattern->prefix.value, name.value, pattern->prefix.length) == 0) {
      return i;
    }
  }
  return -1;
}

__attribute__((nonnull)) static PingReplyId
TgProducer_SelectReply(TgProducer* producer, TgProducerPattern* pattern)
{
  uint32_t rnd = pcg32_random_r(&producer->replyRng);
  return pattern->weight[rnd % pattern->nWeights];
}

__attribute__((nonnull)) static Packet*
TgProducer_RespondData(TgProducer* producer, TgProducerPattern* pattern, TgProducerReply* reply,
                       Packet* npkt)
{
  LpL3* lpl3 = Packet_GetLpL3Hdr(npkt);
  uint64_t token = lpl3->pitToken;
  const LName* name = (const LName*)&Packet_GetInterestHdr(npkt)->name;

  struct rte_mbuf* segs[2];

  segs[0] = rte_pktmbuf_alloc(producer->dataMp);
  if (unlikely(segs[0] == NULL)) {
    ZF_LOGW("dataMp-full");
    ++producer->nAllocError;
    rte_pktmbuf_free(Packet_ToMbuf(npkt));
    return NULL;
  }

  segs[1] = rte_pktmbuf_alloc(producer->indirectMp);
  if (unlikely(segs[1] == NULL)) {
    ZF_LOGW("indirectMp-full");
    ++producer->nAllocError;
    segs[1] = Packet_ToMbuf(npkt);
    rte_pktmbuf_free_bulk(segs, 2);
    return NULL;
  }

  Packet* output = DataGen_Encode(reply->dataGen, segs[0], segs[1], *name);
  rte_pktmbuf_free(Packet_ToMbuf(npkt));
  Packet_GetLpL3Hdr(output)->pitToken = token;
  return output;
}

__attribute__((nonnull)) static Packet*
TgProducer_RespondNack(TgProducer* producer, TgProducerPattern* pattern, TgProducerReply* reply,
                       Packet* npkt)
{
  return Nack_FromInterest(npkt, reply->nackReason);
}

__attribute__((nonnull)) static Packet*
TgProducer_RespondTimeout(TgProducer* producer, TgProducerPattern* pattern, TgProducerReply* reply,
                          Packet* npkt)
{
  rte_pktmbuf_free(Packet_ToMbuf(npkt));
  return NULL;
}

typedef Packet* (*TgProducer_Respond)(TgProducer* producer, TgProducerPattern* pattern,
                                      TgProducerReply* reply, Packet* npkt);

static const TgProducer_Respond TgProducer_RespondJmp[3] = {
  [TGPRODUCER_REPLY_DATA] = TgProducer_RespondData,
  [TGPRODUCER_REPLY_NACK] = TgProducer_RespondNack,
  [TGPRODUCER_REPLY_TIMEOUT] = TgProducer_RespondTimeout,
};

__attribute__((nonnull)) static Packet*
TgProducer_ProcessInterest(TgProducer* producer, Packet* npkt)
{
  uint64_t token = Packet_GetLpL3Hdr(npkt)->pitToken;
  const LName* name = (const LName*)&Packet_GetInterestHdr(npkt)->name;

  int patternId = TgProducer_FindPattern(producer, *name);
  if (unlikely(patternId < 0)) {
    ZF_LOGD(">I dn-token=%016" PRIx64 " no-pattern", token);
    ++producer->nNoMatch;
    if (producer->wantNackNoRoute) {
      return Nack_FromInterest(npkt, NackNoRoute);
    } else {
      rte_pktmbuf_free(Packet_ToMbuf(npkt));
      return NULL;
    }
  }

  TgProducerPattern* pattern = &producer->pattern[patternId];
  uint8_t replyId = TgProducer_SelectReply(producer, pattern);
  TgProducerReply* reply = &pattern->reply[replyId];

  ZF_LOGD(">I dn-token=%016" PRIx64 " pattern=%d reply=%" PRIu8, token, patternId, replyId);
  ++reply->nInterests;
  return TgProducer_RespondJmp[reply->kind](producer, pattern, reply, npkt);
}

int
TgProducer_Run(TgProducer* producer)
{
  struct rte_mbuf* rx[MaxBurstSize];
  Packet* tx[MaxBurstSize];

  while (ThreadStopFlag_ShouldContinue(&producer->stop)) {
    TscTime now = rte_get_tsc_cycles();
    PktQueuePopResult pop = PktQueue_Pop(&producer->rxQueue, rx, MaxBurstSize, now);
    if (unlikely(pop.count == 0)) {
      rte_pause();
      continue;
    }

    uint16_t nTx = 0;
    for (uint16_t i = 0; i < pop.count; ++i) {
      Packet* npkt = Packet_FromMbuf(rx[i]);
      NDNDPDK_ASSERT(Packet_GetType(npkt) == PktInterest);
      tx[nTx] = TgProducer_ProcessInterest(producer, npkt);
      nTx += (int)(tx[nTx] != NULL);
    }

    ZF_LOGD("face=%" PRI_FaceID "nRx=%" PRIu16 " nTx=%" PRIu16, producer->face, pop.count, nTx);
    Face_TxBurst(producer->face, tx, nTx);
  }
  return 0;
}
