#include "fetcher.h"

#include "../core/logger.h"
#include "../ndni/tlv-decoder.h"

N_LOG_INIT(FetchTask);

__attribute__((nonnull)) static inline void
FetchTask_EncodeInterest(FetchTask* fp, FetchThread* fth, struct rte_mbuf* pkt, uint64_t segNum) {
  uint8_t suffix[10];
  LName nameSuffix = {
    .length = Nni_EncodeNameComponent(suffix, TtSegmentNameComponent, segNum),
    .value = suffix,
  };

  uint32_t nonce = pcg32_random_r(&fth->nonceRng);
  Packet* npkt = InterestTemplate_Encode(&fp->tpl, pkt, nameSuffix, nonce);
  LpPitToken_Set(&Packet_GetLpL3Hdr(npkt)->pitToken, sizeof(fp->index), &fp->index);
}

__attribute__((nonnull)) static inline uint32_t
FetchTask_TxBurst(FetchTask* fp, FetchThread* fth) {
  TscTime now = rte_get_tsc_cycles();
  uint64_t segNums[MaxBurstSize];
  size_t count = FetchLogic_TxInterestBurst(&fp->logic, segNums, RTE_DIM(segNums), now);
  if (count == 0) {
    return count;
  }

  struct rte_mbuf* pkts[MaxBurstSize];
  int res = rte_pktmbuf_alloc_bulk(fth->interestMp, pkts, count);
  if (unlikely(res != 0)) {
    N_LOGW("%p interestMp-full", fp);
    return count;
  }

  for (size_t i = 0; i < count; ++i) {
    FetchTask_EncodeInterest(fp, fth, pkts[i], segNums[i]);
  }
  Face_TxBurst(fth->face, (Packet**)pkts, count);
  return count;
}

__attribute__((nonnull)) static inline bool
FetchTask_DecodeData(FetchTask* fp, Packet* npkt, FetchLogicRxData* lpkt) {
  LpL3* lpl3 = Packet_GetLpL3Hdr(npkt);
  lpkt->congMark = lpl3->congMark;

  const PData* data = Packet_GetDataHdr(npkt);
  lpkt->isFinalBlock = data->isFinalBlock;

  const uint8_t* seqNumComp = RTE_PTR_ADD(data->name.value, fp->tpl.prefixL);
  return data->name.length > fp->tpl.prefixL + 1 &&
         // this memcmp checks for SegmentNameComponent TLV-TYPE also
         memcmp(data->name.value, fp->tpl.prefixV, fp->tpl.prefixL + 1) == 0 &&
         Nni_Decode(seqNumComp[1], RTE_PTR_ADD(seqNumComp, 2), &lpkt->segNum);
}

__attribute__((nonnull)) static inline bool
FetchTask_WriteData(FetchThread* fth, FetchTask* fp, Packet* npkt, FetchLogicRxData* lpkt) {
  const PData* data = Packet_GetDataHdr(npkt);
  struct rte_mbuf* pkt = Packet_ToMbuf(npkt);
  struct iovec* iov = (struct iovec*)data->helperScratch;
  if (unlikely(pkt->nb_segs > sizeof(data->helperScratch) / sizeof(iov[0]))) {
    N_LOGW("%p WriteData seg=%" PRIu64 " frags=%" PRIu16 N_LOG_ERROR_STR, fp, lpkt->segNum,
           pkt->nb_segs, "too-many-frags");
    return false;
  }

  struct io_uring_sqe* sqe = Uring_GetSqe(&fth->ur);
  if (unlikely(sqe == NULL)) {
    N_LOGW("%p WriteData seg=%" PRIu64 N_LOG_ERROR_STR, fp, lpkt->segNum, "no-SQE");
    return false;
  }

  int iovcnt = Mbuf_AsIovec(pkt, iov, data->contentOffset, data->contentL);
  io_uring_prep_writev(sqe, fp->fd, iov, iovcnt, lpkt->segNum * fp->segmentLen);
  io_uring_sqe_set_data(sqe, pkt);
  return true;
}

__attribute__((nonnull)) static __rte_always_inline uint32_t
FetchTask_RxBurst(FetchThread* fth, FetchTask* fp, bool wantWrite) {
  TscTime now = rte_get_tsc_cycles();
  Packet* npkts[MaxBurstSize];
  uint32_t nRx = PktQueue_Pop(&fp->queueD, (struct rte_mbuf**)npkts, MaxBurstSize, now).count;
  Packet* discards[MaxBurstSize];
  uint32_t nDiscards = 0;

  FetchLogicRxData lpkts[MaxBurstSize];
  uint32_t count = 0;
  for (uint16_t i = 0; i < nRx; ++i) {
    Packet* npkt = npkts[i];
    FetchLogicRxData* lpkt = &lpkts[count];
    bool ok = FetchTask_DecodeData(fp, npkt, lpkt);
    if (wantWrite && likely(ok)) {
      ok = FetchTask_WriteData(fth, fp, npkt, lpkt);
    }

    if (unlikely(!ok)) {
      discards[nDiscards++] = npkt;
      continue;
    }
    ++count;
  }
  FetchLogic_RxDataBurst(&fp->logic, lpkts, count, now);

  if (wantWrite) {
    if (unlikely(nDiscards > 0)) {
      rte_pktmbuf_free_bulk((struct rte_mbuf**)discards, nDiscards);
    }
  } else {
    rte_pktmbuf_free_bulk((struct rte_mbuf**)npkts, nRx);
  }
  return nRx;
}

__attribute__((nonnull)) static uint32_t
FetchTask_RxBurst_Discard(FetchThread* fth, FetchTask* fp) {
  return FetchTask_RxBurst(fth, fp, false);
}

__attribute__((nonnull)) static uint32_t
FetchTask_RxBurst_Write(FetchThread* fth, FetchTask* fp) {
  return FetchTask_RxBurst(fth, fp, true);
}

typedef uint32_t (*FetchTask_RxBurstFunc)(FetchThread* fth, FetchTask* fp);
const FetchTask_RxBurstFunc FetchTask_RxBurstJmp[] = {
  [false] = FetchTask_RxBurst_Discard,
  [true] = FetchTask_RxBurst_Write,
};

__attribute__((nonnull)) static inline uint32_t
FetchThread_CqBurst(FetchThread* fth) {
  struct io_uring_cqe* cqes[MaxBurstSize];
  uint32_t n = Uring_PeekCqes(&fth->ur, cqes, RTE_DIM(cqes));
  if (n == 0) {
    return 0;
  }

  struct rte_mbuf* discards[MaxBurstSize];
  for (uint32_t i = 0; i < n; ++i) {
    struct io_uring_cqe* cqe = cqes[i];
    if (unlikely(cqe->res < 0)) {
      N_LOGW("%p CQE error" N_LOG_ERROR_ERRNO, fth, cqe->res);
    }
    discards[i] = io_uring_cqe_get_data(cqe);
  }
  Uring_SeenCqes(&fth->ur, n);

  rte_pktmbuf_free_bulk(discards, n);
  return n;
}

int
FetchThread_Run(FetchThread* fth) {
  bool ok = Uring_Init(&fth->ur, fth->uringCapacity);
  if (unlikely(!ok)) {
    return 1;
  }

  uint32_t nProcessed = 0;
  while (ThreadCtrl_Continue(fth->ctrl, nProcessed)) {
    rcu_quiescent_state();
    rcu_read_lock();
    FetchTask* fp;
    struct cds_hlist_node* pos;
    cds_hlist_for_each_entry_rcu (fp, pos, &fth->tasksHead, fthNode) {
      MinSched_Trigger(fp->logic.sched);
      nProcessed += FetchTask_TxBurst(fp, fth);
      nProcessed += FetchTask_RxBurstJmp[fp->fd >= 0](fth, fp);
    }

    Uring_Submit(&fth->ur, fth->uringWaitLbound, MaxBurstSize);
    nProcessed += FetchThread_CqBurst(fth);
    rcu_read_unlock();
  }

  Uring_Free(&fth->ur);
  return 0;
}
