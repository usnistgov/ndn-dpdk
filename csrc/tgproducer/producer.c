#include "producer.h"

#include "../core/logger.h"

N_LOG_INIT(TgProducer);

__attribute__((nonnull)) static int
TgProducer_FindPattern(TgProducer* p, LName name)
{
  for (uint16_t i = 0; i < p->nPatterns; ++i) {
    TgProducerPattern* pattern = &p->pattern[i];
    if (pattern->prefix.length <= name.length &&
        memcmp(pattern->prefix.value, name.value, pattern->prefix.length) == 0) {
      return i;
    }
  }
  return -1;
}

__attribute__((nonnull)) static PingReplyId
TgProducer_SelectReply(TgProducer* p, TgProducerPattern* pattern)
{
  uint32_t rnd = pcg32_random_r(&p->replyRng);
  return pattern->weight[rnd % pattern->nWeights];
}

__attribute__((nonnull)) static Packet*
TgProducer_RespondData(TgProducer* p, TgProducerPattern* pattern, TgProducerReply* reply,
                       Packet* npkt)
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
TgProducer_RespondNack(TgProducer* p, TgProducerPattern* pattern, TgProducerReply* reply,
                       Packet* npkt)
{
  return Nack_FromInterest(npkt, reply->nackReason, &p->mp, Face_PacketTxAlign(p->face));
}

__attribute__((nonnull)) static Packet*
TgProducer_RespondTimeout(TgProducer* p, TgProducerPattern* pattern, TgProducerReply* reply,
                          Packet* npkt)
{
  rte_pktmbuf_free(Packet_ToMbuf(npkt));
  return NULL;
}

typedef Packet* (*TgProducer_Respond)(TgProducer* p, TgProducerPattern* pattern,
                                      TgProducerReply* reply, Packet* npkt);

static const TgProducer_Respond TgProducer_RespondJmp[3] = {
  [TGPRODUCER_REPLY_DATA] = TgProducer_RespondData,
  [TGPRODUCER_REPLY_NACK] = TgProducer_RespondNack,
  [TGPRODUCER_REPLY_TIMEOUT] = TgProducer_RespondTimeout,
};

__attribute__((nonnull)) static Packet*
TgProducer_ProcessInterest(TgProducer* p, Packet* npkt)
{
  uint64_t token = Packet_GetLpL3Hdr(npkt)->pitToken;
  const LName* name = (const LName*)&Packet_GetInterestHdr(npkt)->name;

  int patternId = TgProducer_FindPattern(p, *name);
  if (unlikely(patternId < 0)) {
    N_LOGD(">I dn-token=%016" PRIx64 " no-pattern", token);
    ++p->nNoMatch;
    if (p->wantNackNoRoute) {
      return Nack_FromInterest(npkt, NackNoRoute, &p->mp, Face_PacketTxAlign(p->face));
    } else {
      rte_pktmbuf_free(Packet_ToMbuf(npkt));
      return NULL;
    }
  }

  TgProducerPattern* pattern = &p->pattern[patternId];
  uint8_t replyId = TgProducer_SelectReply(p, pattern);
  TgProducerReply* reply = &pattern->reply[replyId];

  N_LOGD(">I dn-token=%016" PRIx64 " pattern=%d reply=%" PRIu8, token, patternId, replyId);
  ++reply->nInterests;
  return TgProducer_RespondJmp[reply->kind](p, pattern, reply, npkt);
}

int
TgProducer_Run(TgProducer* p)
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
      tx[nTx] = TgProducer_ProcessInterest(p, npkt);
      nTx += (int)(tx[nTx] != NULL);
    }

    N_LOGD("burst face=%" PRI_FaceID "nRx=%" PRIu16 " nTx=%" PRIu16, p->face, pop.count, nTx);
    Face_TxBurst(p->face, tx, nTx);
  }
  return 0;
}
