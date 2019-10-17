#include "crypto.h"

#include "../../core/logger.h"

INIT_ZF_LOG(FwCrypto);

#define FW_CRYPTO_BURST_SIZE 16

static void
FwCrypto_Input(FwCrypto* fwc)
{
  Packet* npkts[FW_CRYPTO_BURST_SIZE];
  uint16_t nDeq = rte_ring_dequeue_burst(
    fwc->input, (void**)npkts, FW_CRYPTO_BURST_SIZE, NULL);
  if (nDeq == 0) {
    return;
  }

  struct rte_crypto_op* ops[FW_CRYPTO_BURST_SIZE];
  uint16_t nAlloc = rte_crypto_op_bulk_alloc(
    fwc->opPool, RTE_CRYPTO_OP_TYPE_SYMMETRIC, ops, nDeq);
  if (unlikely(nAlloc == 0)) {
    ZF_LOGW("fwc=%p rte_crypto_op_bulk_alloc fail", fwc);
    FreeMbufs((struct rte_mbuf**)npkts, nDeq);
    return;
  }

  for (uint16_t i = 0; i < nDeq; ++i) {
    DataDigest_Prepare(npkts[i], ops[i]);
  }

  uint16_t nEnq = rte_cryptodev_enqueue_burst(fwc->devId, fwc->qpId, ops, nDeq);
  for (uint16_t i = nEnq; i < nDeq; ++i) {
    Packet* npkt = DataDigest_Finish(ops[i]);
    RTE_ASSERT(npkt == NULL);
    RTE_SET_USED(npkt);
  }
}

static void
FwCrypto_Output(FwCrypto* fwc)
{
  struct rte_crypto_op* ops[FW_CRYPTO_BURST_SIZE];
  uint16_t nDeq = rte_cryptodev_dequeue_burst(
    fwc->devId, fwc->qpId, ops, FW_CRYPTO_BURST_SIZE);

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
    LpL3* lpl3 = Packet_GetLpL3Hdr(npkt);
    FwInput_DispatchByToken(fwc->output, npkt, lpl3->pitToken);
  }
}

void
FwCrypto_Run(FwCrypto* fwc)
{
  ZF_LOGI("fwc=%p input=%p pool=%p cryptodev=%" PRIu8 "-%" PRIu16 " output=%p",
          fwc,
          fwc->input,
          fwc->opPool,
          fwc->devId,
          fwc->qpId,
          fwc->output);
  while (ThreadStopFlag_ShouldContinue(&fwc->stop)) {
    FwCrypto_Output(fwc);
    FwCrypto_Input(fwc);
  }
}
