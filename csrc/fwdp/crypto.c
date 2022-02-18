#include "crypto.h"

#include "../core/logger.h"

N_LOG_INIT(FwCrypto);

#define FW_CRYPTO_BURST_SIZE 16

__attribute__((nonnull)) static uint16_t
FwCrypto_Input(FwCrypto* fwc)
{
  Packet* npkts[FW_CRYPTO_BURST_SIZE];
  uint16_t nDeq = rte_ring_dequeue_burst(fwc->input, (void**)npkts, FW_CRYPTO_BURST_SIZE, NULL);
  if (nDeq == 0) {
    return nDeq;
  }

  struct rte_crypto_op* ops[FW_CRYPTO_BURST_SIZE];
  uint16_t nAlloc = rte_crypto_op_bulk_alloc(fwc->opPool, RTE_CRYPTO_OP_TYPE_SYMMETRIC, ops, nDeq);
  if (unlikely(nAlloc == 0)) {
    N_LOGW("rte_crypto_op_bulk_alloc fail fwc=%p", fwc);
    rte_pktmbuf_free_bulk((struct rte_mbuf**)npkts, nDeq);
    return nDeq;
  }

  for (uint16_t i = 0; i < nDeq; ++i) {
    Packet* npkt = npkts[i];
    DataDigest_Prepare(npkt, ops[i]);
  }

  fwc->nDrops += DataDigest_Enqueue(fwc->cqp, ops, nDeq);
  return nDeq;
}

__attribute__((nonnull)) static uint16_t
FwCrypto_Output(FwCrypto* fwc, CryptoQueuePair cqp)
{
  struct rte_crypto_op* ops[FW_CRYPTO_BURST_SIZE];
  uint16_t nDeq = rte_cryptodev_dequeue_burst(cqp.dev, cqp.qp, ops, FW_CRYPTO_BURST_SIZE);

  Packet* npkts[FW_CRYPTO_BURST_SIZE];
  uint16_t nFinish = 0;
  for (uint16_t i = 0; i < nDeq; ++i) {
    npkts[nFinish] = DataDigest_Finish(ops[i]);
    if (likely(npkts[nFinish] != NULL)) {
      ++nFinish;
    }
  }

  struct rte_mbuf* drops[FW_CRYPTO_BURST_SIZE];
  uint16_t nDrops = 0;
  for (uint16_t i = 0; i < nFinish; ++i) {
    Packet* npkt = npkts[i];
    bool accepted = InputDemux_Dispatch(&fwc->output, npkt);
    if (unlikely(!accepted)) {
      drops[nDrops++] = Packet_ToMbuf(npkt);
    }
  }

  if (unlikely(nDrops > 0)) {
    rte_pktmbuf_free_bulk(drops, nDrops);
  }
  return nDeq;
}

void
FwCrypto_Run(FwCrypto* fwc)
{
  N_LOGI("Run fwc=%p input=%p pool=%p cryptodev=%" PRIu8 "-%" PRIu16, fwc, fwc->input, fwc->opPool,
         fwc->cqp.dev, fwc->cqp.qp);
  uint16_t nProcessed = 0;
  while (ThreadCtrl_Continue(fwc->ctrl, nProcessed)) {
    nProcessed += FwCrypto_Output(fwc, fwc->cqp);
    nProcessed += FwCrypto_Input(fwc);
  }
}
