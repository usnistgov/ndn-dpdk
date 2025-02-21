#include "crypto.h"

#include "../core/logger.h"

N_LOG_INIT(FwCrypto);

#define FW_CRYPTO_BURST_SIZE 16

__attribute__((nonnull)) static uint16_t
FwCrypto_Input(FwCrypto* fwc) {
  Packet* npkts[FW_CRYPTO_BURST_SIZE];
  uint16_t nDeq = rte_ring_dequeue_burst(fwc->input, (void**)npkts, FW_CRYPTO_BURST_SIZE, NULL);
  if (nDeq == 0) {
    return 0;
  }

  struct rte_crypto_op* ops[FW_CRYPTO_BURST_SIZE];
  for (uint16_t i = 0; i < nDeq; ++i) {
    Packet* npkt = npkts[i];
    ops[i] = DataDigest_Prepare(&fwc->cqp, npkt);
  }

  uint16_t nRej = DataDigest_Enqueue(&fwc->cqp, ops, nDeq);
  if (unlikely(nRej > 0)) {
    fwc->nDrops += nRej;
    rte_pktmbuf_free_bulk((struct rte_mbuf**)&npkts[nDeq - nRej], nRej);
  }
  return nDeq;
}

__attribute__((nonnull)) static uint16_t
FwCrypto_Output(FwCrypto* fwc, CryptoQueuePair cqp) {
  struct rte_crypto_op* ops[FW_CRYPTO_BURST_SIZE];
  uint16_t nDeq = rte_cryptodev_dequeue_burst(cqp.dev, cqp.qp, ops, FW_CRYPTO_BURST_SIZE);

  Packet* npkts[FW_CRYPTO_BURST_SIZE];
  uint16_t nFinish = 0;
  struct rte_mbuf* drops[FW_CRYPTO_BURST_SIZE];
  uint16_t nDrops = 0;

  for (uint16_t i = 0; i < nDeq; ++i) {
    if (likely(DataDigest_Finish(ops[i], &npkts[nFinish]))) {
      ++nFinish;
    } else {
      drops[nDrops++] = Packet_ToMbuf(npkts[nFinish]);
    }
  }

  if (nFinish > 0) {
    uint64_t rejectMask = InputDemux_Dispatch(&fwc->output, npkts, nFinish);
    InputDemux_FreeRejected(drops, &nDrops, npkts, rejectMask);
  }

  if (unlikely(nDrops > 0)) {
    rte_pktmbuf_free_bulk(drops, nDrops);
  }
  return nDeq;
}

void
FwCrypto_Run(FwCrypto* fwc) {
  N_LOGI("Run fwc=%p input=%p cryptodev=%" PRIu8 "-%" PRIu16, fwc, fwc->input, fwc->cqp.dev,
         fwc->cqp.qp);
  uint16_t nProcessed = 0;
  while (ThreadCtrl_Continue(fwc->ctrl, nProcessed)) {
    nProcessed += FwCrypto_Output(fwc, fwc->cqp);
    nProcessed += FwCrypto_Input(fwc);
  }
}
