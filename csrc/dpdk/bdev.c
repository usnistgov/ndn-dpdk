#include "bdev.h"

void* BdevFiller_ = NULL;

__attribute__((nonnull)) static inline void
Bdev_Complete(struct spdk_bdev_io* io, bool success, void* req0)
{
  spdk_bdev_free_io(io);
  NULLize(io);

  BdevRequest* req = (BdevRequest*)req0;
  int res = success ? 0 : EIO;
  req->cb(req, res);
}

__attribute__((nonnull)) static void
Bdev_ReadSuccess(BdevRequest* req)
{
  struct rte_mbuf* pkt = req->pkt;
  BdevStoredPacket* sp = req->sp;
  pkt->pkt_len = sp->pktLen;
  pkt->data_len = sp->pktLen;
  if (sp->saveTotal == sp->pktLen) {
    return;
  }

  uint8_t* dst = rte_pktmbuf_mtod(pkt, uint8_t*);
  const uint8_t* src = dst;
  for (int i = 0; i < BdevMaxMbufSegs; ++i) {
    uint16_t saveLen = sp->saveLen[i];
    if (saveLen == 0) {
      break;
    }

    uint16_t headLen = sp->headTail[i] >> 4;
    uint16_t headTailLen = sp->headTail[i] & 0x0F;
    uint16_t segLen = saveLen - headTailLen;

    if (i == 0 && sp->saveLen[1] == 0) {
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
Bdev_ReadComplete(struct spdk_bdev_io* io, bool success, void* req0)
{
  if (likely(success)) {
    Bdev_ReadSuccess(req0);
  }
  Bdev_Complete(io, success, req0);
}

void
Bdev_ReadPacket(Bdev* bd, struct spdk_io_channel* ch, uint64_t blockOffset, BdevRequest* req)
{
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

bool
Bdev_WritePrepare(Bdev* bd, struct rte_mbuf* pkt, BdevStoredPacket* sp)
{
  if (pkt->pkt_len == 0 || pkt->pkt_len >= UINT16_MAX || pkt->nb_segs > BdevMaxMbufSegs) {
    return false;
  }

  sp->pktLen = pkt->pkt_len;
  if (!bd->dwordAlign) {
    sp->saveTotal = sp->pktLen;
    return true;
  }

  uint32_t saveTotal = 0;
  int i = 0;
  for (struct rte_mbuf* m = pkt; m != NULL; m = m->next) {
    NDNDPDK_ASSERT(m->data_len != 0);
    uint16_t headLen = m->data_off & 0x03;
    uint16_t saveLen = RTE_ALIGN_CEIL(headLen + m->data_len, 4);
    uint16_t headTailLen = saveLen - m->data_len;
    saveTotal += saveLen;
    sp->saveLen[i] = saveLen;
    sp->headTail[i] = (headLen << 4) | headTailLen;
    ++i;
  }
  if (likely(i < BdevMaxMbufSegs)) {
    sp->saveLen[i] = 0;
  }

  if (saveTotal >= UINT16_MAX) {
    return false;
  }
  sp->saveTotal = saveTotal;
  return true;
}

void
Bdev_WritePacket(Bdev* bd, struct spdk_io_channel* ch, uint64_t blockOffset, BdevRequest* req)
{
  BdevStoredPacket* sp = req->sp;
  uint32_t blockCount = BdevStoredPacket_ComputeBlockCount(sp);
  uint32_t totalLen = blockCount * BdevBlockSize;

  struct rte_mbuf* pkt = req->pkt;
  struct rte_mbuf* lastSeg = pkt;
  int iovcnt = 0;
  for (struct rte_mbuf* seg = pkt; seg != NULL; seg = seg->next) {
    if (bd->dwordAlign) {
      uint16_t headLen = sp->headTail[iovcnt] >> 4;
      req->iov_[iovcnt].iov_base = RTE_PTR_SUB(rte_pktmbuf_mtod(seg, void*), headLen);
      req->iov_[iovcnt].iov_len = sp->saveLen[iovcnt];
    } else {
      req->iov_[iovcnt].iov_base = rte_pktmbuf_mtod(seg, void*);
      req->iov_[iovcnt].iov_len = seg->data_len;
    }
    ++iovcnt;
    lastSeg = seg;
  }

  size_t fillerLen = totalLen - sp->saveTotal;
  if (likely(
        RTE_PTR_ADD(req->iov_[iovcnt - 1].iov_base, req->iov_[iovcnt - 1].iov_len + fillerLen) <=
        RTE_PTR_ADD(lastSeg->buf_addr, lastSeg->buf_len))) {
    req->iov_[iovcnt - 1].iov_len += fillerLen;
  } else {
    req->iov_[iovcnt].iov_base = BdevFiller_;
    req->iov_[iovcnt].iov_len = fillerLen;
    ++iovcnt;
  }

  int res = spdk_bdev_writev_blocks(bd->desc, ch, req->iov_, iovcnt, blockOffset, blockCount,
                                    Bdev_Complete, req);
  if (unlikely(res != 0)) {
    req->cb(req, res);
  }
}
