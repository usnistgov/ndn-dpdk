#include "bdev.h"

uint8_t* BdevFiller_ = NULL;

__attribute__((nonnull)) static __rte_always_inline uint16_t
BdevStoredPacket_ComputeHeadTail(struct rte_mbuf* m, uint16_t* headLen, uint16_t* saveLen,
                                 uint8_t* headTail) {
  NDNDPDK_ASSERT(m->data_len != 0);
  *headLen = m->data_off & 0x03;
  *saveLen = RTE_ALIGN_CEIL(*headLen + m->data_len, 4);
  uint16_t headTailLen = *saveLen - m->data_len;
  *headTail = (*headLen << 4) | headTailLen;
  return *saveLen;
}

__attribute__((nonnull)) static __rte_always_inline void
BdevStoredPacket_SplitHeadTail(uint16_t saveLen, uint8_t headTail, uint16_t* headLen,
                               uint16_t* segLen) {
  *headLen = headTail >> 4;
  uint16_t headTailLen = headTail & 0x0F;
  *segLen = saveLen - headTailLen;
}

__attribute__((nonnull)) static inline void
Bdev_Complete(struct spdk_bdev_io* io, int res, void* req0) {
  spdk_bdev_free_io(io);
  NULLize(io);

  BdevRequest* req = (BdevRequest*)req0;
  req->cb(req, res);
}

__attribute__((nonnull)) static void
Bdev_ReadSuccess(BdevRequest* req) {
  struct rte_mbuf* pkt = req->pkt;
  BdevStoredPacket* sp = req->sp;
  pkt->pkt_len = sp->pktLen;
  pkt->data_len = sp->pktLen;
  if (sp->saveTotal == sp->pktLen) { // no head or tail
    return;
  }

  uint8_t* dst = rte_pktmbuf_mtod(pkt, uint8_t*);
  const uint8_t* src = dst;
  for (int i = 0; i < BdevMaxMbufSegs; ++i) {
    uint16_t saveLen = sp->saveLen[i];
    if (saveLen == 0) {
      break;
    }

    uint16_t headLen, segLen;
    BdevStoredPacket_SplitHeadTail(saveLen, sp->headTail[i], &headLen, &segLen);

    if (i == 0 && sp->saveLen[1] == 0) { // single segment
      pkt->data_off += headLen;
      return;
    }

    const uint8_t* src1 = RTE_PTR_ADD(src, headLen);
    if (likely(dst != src1)) {
      memmove(dst, src1, segLen);
    }
    dst = RTE_PTR_ADD(dst, segLen);
    src = RTE_PTR_ADD(src, saveLen);
  }

  NDNDPDK_ASSERT(rte_pktmbuf_mtod_offset(pkt, uint8_t*, sp->pktLen) == dst);
  NDNDPDK_ASSERT(rte_pktmbuf_mtod_offset(pkt, const uint8_t*, sp->saveTotal) == src);
}

__attribute__((nonnull)) static void
Bdev_ReadComplete(struct spdk_bdev_io* io, bool success, void* req0) {
  if (likely(success)) {
    Bdev_ReadSuccess(req0);
  }
  Bdev_Complete(io, success ? 0 : EIO, req0);
}

void
Bdev_ReadPacket(Bdev* bd, struct spdk_io_channel* ch, uint64_t blockOffset, BdevRequest* req) {
  uint32_t blockCount = BdevStoredPacket_ComputeBlockCount(req->sp);
  uint32_t totalLen = blockCount * BdevBlockSize;

  struct rte_mbuf* pkt = req->pkt;
  NDNDPDK_ASSERT(RTE_MBUF_DIRECT(pkt) && rte_pktmbuf_is_contiguous(pkt) &&
                 rte_mbuf_refcnt_read(pkt) == 1);

  void* first = RTE_PTR_ALIGN_CEIL(pkt->buf_addr, bd->bufAlign);
  void* last = RTE_PTR_ALIGN_FLOOR(RTE_PTR_ADD(pkt->buf_addr, pkt->buf_len), bd->bufAlign);
  if (unlikely(RTE_PTR_ADD(first, totalLen) > last)) {
    req->cb(req, ENOBUFS);
    return;
  }

  pkt->data_off = RTE_PTR_DIFF(first, pkt->buf_addr);
  int res =
    spdk_bdev_read_blocks(bd->desc, ch, first, blockOffset, blockCount, Bdev_ReadComplete, req);
  if (unlikely(res != 0)) {
    req->cb(req, res);
  }
}

__attribute__((nonnull)) static inline int
BdevWrite_AppendFiller(BdevRequest* req, uint32_t totalLen, struct rte_mbuf* lastSeg, int iovcnt) {
  size_t fillerLen = totalLen - req->sp->saveTotal;
  if (likely(
        RTE_PTR_ADD(req->iov_[iovcnt - 1].iov_base, req->iov_[iovcnt - 1].iov_len + fillerLen) <=
        RTE_PTR_ADD(lastSeg->buf_addr, lastSeg->buf_len))) {
    req->iov_[iovcnt - 1].iov_len += fillerLen;
  } else {
    req->iov_[iovcnt].iov_base = BdevFiller_;
    req->iov_[iovcnt].iov_len = fillerLen;
    ++iovcnt;
  }
  return iovcnt;
}

__attribute__((nonnull)) static bool
BdevWrite_SimplePrepare(__rte_unused Bdev* bd, __rte_unused struct rte_mbuf* pkt,
                        BdevStoredPacket* sp) {
  sp->saveTotal = sp->pktLen;
  return true;
}

__attribute__((nonnull)) static int
BdevWrite_SimpleIov(__rte_unused Bdev* bd, BdevRequest* req, uint32_t totalLen) {
  struct rte_mbuf* pkt = req->pkt;
  struct rte_mbuf* lastSeg = pkt;
  int iovcnt = 0;
  for (struct rte_mbuf* seg = pkt; seg != NULL; seg = seg->next) {
    req->iov_[iovcnt] = (struct iovec){
      .iov_base = rte_pktmbuf_mtod(seg, void*),
      .iov_len = seg->data_len,
    };
    ++iovcnt;
    lastSeg = seg;
  }
  iovcnt = BdevWrite_AppendFiller(req, totalLen, lastSeg, iovcnt);
  return iovcnt;
}

__attribute__((nonnull)) static bool
BdevWrite_DwordPrepare(__rte_unused Bdev* bd, struct rte_mbuf* pkt, BdevStoredPacket* sp) {
  sp->saveTotal = 0;
  int i = 0;
  for (struct rte_mbuf* m = pkt; m != NULL; m = m->next) {
    uint16_t headLen;
    sp->saveTotal +=
      BdevStoredPacket_ComputeHeadTail(m, &headLen, &sp->saveLen[i], &sp->headTail[i]);
    ++i;
  }
  if (likely(i < BdevMaxMbufSegs)) {
    sp->saveLen[i] = 0;
  }

  if (sp->saveTotal >= UINT16_MAX) {
    return false;
  }
  return true;
}

__attribute__((nonnull)) static int
BdevWrite_DwordIov(__rte_unused Bdev* bd, BdevRequest* req, uint32_t totalLen) {
  BdevStoredPacket* sp = req->sp;
  struct rte_mbuf* pkt = req->pkt;
  struct rte_mbuf* lastSeg = pkt;
  int i = 0;
  for (struct rte_mbuf* seg = pkt; seg != NULL; seg = seg->next) {
    uint16_t headLen, segLen;
    BdevStoredPacket_SplitHeadTail(sp->saveLen[i], sp->headTail[i], &headLen, &segLen);
    req->iov_[i] = (struct iovec){
      .iov_base = rte_pktmbuf_mtod_offset(seg, void*, -headLen),
      .iov_len = sp->saveLen[i],
    };
    ++i;
    lastSeg = seg;
  }
  i = BdevWrite_AppendFiller(req, totalLen, lastSeg, i);
  return i;
}

__attribute__((nonnull)) static inline bool
BdevWrite_ContigHasTotalBuf(struct rte_mbuf* m, BdevStoredPacket* sp, uint16_t* headLen) {
  uint32_t totalLen = BdevStoredPacket_ComputeBlockCount(sp) * BdevBlockSize;
  *headLen = 0;
  uint16_t segLen = m->data_len;
  if (sp->saveTotal != sp->pktLen) {
    BdevStoredPacket_SplitHeadTail(sp->saveLen[0], sp->headTail[0], headLen, &segLen);
  }
  return m->data_off - *headLen + totalLen <= m->buf_len;
}

__attribute__((nonnull)) static bool
BdevWrite_ContigPrepare(Bdev* bd, struct rte_mbuf* pkt, BdevStoredPacket* sp) {
  uint16_t headLen = 0;
  if (pkt->nb_segs == 1 && BdevWrite_DwordPrepare(bd, pkt, sp) &&
      BdevWrite_ContigHasTotalBuf(pkt, sp, &headLen)) {
    return true;
  }
  return BdevWrite_SimplePrepare(bd, pkt, sp);
}

__attribute__((nonnull)) static int
BdevWrite_ContigIov(Bdev* bd, BdevRequest* req, uint32_t totalLen) {
  BdevStoredPacket* sp = req->sp;
  struct rte_mbuf* pkt = req->pkt;
  uint16_t headLen = 0;
  if (pkt->nb_segs == 1 && BdevWrite_ContigHasTotalBuf(pkt, sp, &headLen)) {
    req->iov_[0] = (struct iovec){
      .iov_base = rte_pktmbuf_mtod_offset(pkt, void*, -headLen),
      .iov_len = totalLen,
    };
    return 1;
  }

  req->bounce_ = rte_pktmbuf_alloc(bd->bounceMp);
  if (unlikely(req->bounce_ == NULL)) {
    return -ENOMEM;
  }

  void* dst = RTE_PTR_ALIGN_CEIL(req->bounce_->buf_addr, bd->bufAlign);
  Mbuf_ReadTo(pkt, 0, pkt->pkt_len, dst);
  req->iov_[0] = (struct iovec){
    .iov_base = dst,
    .iov_len = totalLen,
  };
  return 1;
}

static const struct {
  bool (*prepare)(Bdev* bd, struct rte_mbuf* pkt, BdevStoredPacket* sp);
  int (*writeIov)(Bdev* bd, BdevRequest* req, uint32_t totalLen);
} BdevWriteOps[] = {
  [BdevWriteModeSimple] =
    {
      .prepare = BdevWrite_SimplePrepare,
      .writeIov = BdevWrite_SimpleIov,
    },
  [BdevWriteModeDwordAlign] =
    {
      .prepare = BdevWrite_DwordPrepare,
      .writeIov = BdevWrite_DwordIov,
    },
  [BdevWriteModeContiguous] =
    {
      .prepare = BdevWrite_ContigPrepare,
      .writeIov = BdevWrite_ContigIov,
    },
};

bool
Bdev_WritePrepare(Bdev* bd, struct rte_mbuf* pkt, BdevStoredPacket* sp) {
  if (pkt->pkt_len == 0 || pkt->pkt_len >= UINT16_MAX || pkt->nb_segs > BdevMaxMbufSegs) {
    return false;
  }

  sp->pktLen = pkt->pkt_len;
  return BdevWriteOps[bd->writeMode].prepare(bd, pkt, sp);
}

__attribute__((nonnull)) static void
Bdev_WriteComplete(struct spdk_bdev_io* io, bool success, void* req0) {
  BdevRequest* req = (BdevRequest*)req0;
  if (req->bounce_ != NULL) {
    rte_pktmbuf_free(req->bounce_);
    NULLize(req->bounce_);
  }
  Bdev_Complete(io, success ? 0 : EIO, req0);
}

void
Bdev_WritePacket(Bdev* bd, struct spdk_io_channel* ch, uint64_t blockOffset, BdevRequest* req) {
  BdevStoredPacket* sp = req->sp;
  uint32_t blockCount = BdevStoredPacket_ComputeBlockCount(sp);
  uint32_t totalLen = blockCount * BdevBlockSize;

  int iovcnt = BdevWriteOps[bd->writeMode].writeIov(bd, req, totalLen);
  if (unlikely(iovcnt < 0)) {
    req->cb(req, -iovcnt);
    return;
  }

  int res = spdk_bdev_writev_blocks(bd->desc, ch, req->iov_, iovcnt, blockOffset, blockCount,
                                    Bdev_WriteComplete, req);
  if (unlikely(res != 0)) {
    req->cb(req, res);
  }
}
