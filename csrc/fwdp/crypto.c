#include "crypto.h"

#include "../core/logger.h"

INIT_ZF_LOG(FwCrypto);

#define FW_CRYPTO_BURST_SIZE 16

static void
FwCrypto_InputEnqueue(FwCrypto* fwc, const CryptoQueuePair* cqp, struct rte_crypto_op** ops,
                      uint16_t count)
{
  if (unlikely(count == 0)) {
    return;
  }

  uint16_t nEnq = rte_cryptodev_enqueue_burst(cqp->dev, cqp->qp, ops, count);
  for (uint16_t i = nEnq; i < count; ++i) {
    Packet* npkt = DataDigest_Finish(ops[i]);
    NDNDPDK_ASSERT(npkt == NULL);
    ++fwc->nDrops;
  }
}

static void
FwCrypto_Input(FwCrypto* fwc)
{
  Packet* npkts[FW_CRYPTO_BURST_SIZE];
  uint16_t nDeq = rte_ring_dequeue_burst(fwc->input, (void**)npkts, FW_CRYPTO_BURST_SIZE, NULL);
  if (nDeq == 0) {
    return;
  }

  struct rte_crypto_op* ops[FW_CRYPTO_BURST_SIZE];
  uint16_t nAlloc = rte_crypto_op_bulk_alloc(fwc->opPool, RTE_CRYPTO_OP_TYPE_SYMMETRIC, ops, nDeq);
  if (unlikely(nAlloc == 0)) {
    ZF_LOGW("fwc=%p rte_crypto_op_bulk_alloc fail", fwc);
    rte_pktmbuf_free_bulk((struct rte_mbuf**)npkts, nDeq);
    return;
  }

  uint16_t posS = 0, posM = nDeq;
  for (uint16_t i = 0; i < nDeq; ++i) {
    Packet* npkt = npkts[i];
    struct rte_mbuf* pkt = Packet_ToMbuf(npkts[i]);
    if (likely(pkt->nb_segs == 1)) {
      DataDigest_Prepare(npkt, ops[posS++]);
    } else {
      DataDigest_Prepare(npkt, ops[--posM]);
    }
  }
  NDNDPDK_ASSERT(posS == posM);

  FwCrypto_InputEnqueue(fwc, &fwc->singleSeg, ops, posS);
  FwCrypto_InputEnqueue(fwc, &fwc->multiSeg, &ops[posM], nDeq - posM);
}

static void
FwCrypto_Output(FwCrypto* fwc, const CryptoQueuePair* cqp)
{
  struct rte_crypto_op* ops[FW_CRYPTO_BURST_SIZE];
  uint16_t nDeq = rte_cryptodev_dequeue_burst(cqp->dev, cqp->qp, ops, FW_CRYPTO_BURST_SIZE);

  Packet* npkts[FW_CRYPTO_BURST_SIZE];
  uint16_t nFinish = 0;
  for (uint16_t i = 0; i < nDeq; ++i) {
    npkts[nFinish] = DataDigest_Finish(ops[i]);
    if (likely(npkts[nFinish] != NULL)) {
      ++nFinish;
    }
  }

  for (uint16_t i = 0; i < nFinish; ++i) {
    Packet* npkt = npkts[i];
    PData* data = Packet_GetDataHdr(npkt);
    InputDemux_Dispatch(&fwc->output, npkt, &data->name);
  }
}

void
FwCrypto_Run(FwCrypto* fwc)
{
  ZF_LOGI("fwc=%p input=%p pool=%p cryptodev-single=%" PRIu8 "-%" PRIu16 " cryptodev-multi=%" PRIu8
          "-%" PRIu16,
          fwc, fwc->input, fwc->opPool, fwc->singleSeg.dev, fwc->singleSeg.qp, fwc->multiSeg.dev,
          fwc->multiSeg.qp);
  while (ThreadStopFlag_ShouldContinue(&fwc->stop)) {
    FwCrypto_Output(fwc, &fwc->singleSeg);
    FwCrypto_Output(fwc, &fwc->multiSeg);
    FwCrypto_Input(fwc);
  }
}
