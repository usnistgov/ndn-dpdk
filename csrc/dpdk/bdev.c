#include "bdev.h"

void* BdevFillers_ = NULL;

__attribute__((nonnull)) static __rte_always_inline int
Bdev_PrepareIovec(Bdev* bd, BdevRequest* req, struct rte_mbuf* pkt, uint32_t totalLen, bool isRead)
{
  if (unlikely(totalLen == 0 || pkt->nb_segs > BdevMaxMbufSegs)) {
    return -EMSGSIZE;
  }

  if (isRead && bd->bufAlign > 1 &&
      likely(RTE_MBUF_DIRECT(pkt) && rte_pktmbuf_is_contiguous(pkt) &&
             rte_mbuf_refcnt_read(pkt) == 1)) {
    void* first = RTE_PTR_ALIGN_CEIL(pkt->buf_addr, bd->bufAlign);
    void* last = RTE_PTR_ALIGN_FLOOR(RTE_PTR_ADD(pkt->buf_addr, pkt->buf_len), bd->bufAlign);
    if (RTE_PTR_ADD(first, totalLen) <= last) {
      pkt->data_off = RTE_PTR_DIFF(first, pkt->buf_addr);
      req->iov[0].iov_base = first;
      req->iov[0].iov_len = totalLen;
      return 1;
    }
    // In forwarder use case, it's OK to use the lowest aligned buffer address without leaving
    // more headroom, because the Data packet is stored in Content Store, and outgoing packets
    // would be indirect mbufs referencing this mbuf, not prepending to this mbuf directly.
    // To test the effectiveness of this optimization, gdb a test case and count invocations of
    // libspdk_bdev _copy_buf_to_iovs function, https://stackoverflow.com/a/41984957
  }

  int iovcnt = 0;
  struct rte_mbuf* lastSeg = pkt;
  for (struct rte_mbuf* seg = pkt; seg != NULL; seg = seg->next) {
    if (unlikely(seg->data_len == 0)) {
      continue;
    }
    req->iov[iovcnt].iov_base = rte_pktmbuf_mtod(seg, void*);
    req->iov[iovcnt].iov_len = seg->data_len;
    ++iovcnt;
    lastSeg = seg;
  }

  size_t fillerLen = totalLen - pkt->pkt_len;
  if (likely(rte_pktmbuf_tailroom(lastSeg) >= fillerLen)) {
    req->iov[iovcnt - 1].iov_len += fillerLen;
    return iovcnt;
  }

  req->iov[iovcnt].iov_base = isRead ? RTE_PTR_ADD(BdevFillers_, BdevFillerLen_) : BdevFillers_;
  req->iov[iovcnt].iov_len = fillerLen;
  ++iovcnt;
  return iovcnt;
}

__attribute__((nonnull)) static void
Bdev_Complete(struct spdk_bdev_io* io, bool success, void* req0)
{
  spdk_bdev_free_io(io);
  NULLize(io);

  BdevRequest* req = (BdevRequest*)req0;
  int res = success ? 0 : EIO;
  req->cb(req, res);
}

void
Bdev_ReadPacket(Bdev* bd, struct spdk_io_channel* ch, struct rte_mbuf* pkt, uint64_t blockOffset,
                BdevRequestCb cb, BdevRequest* req)
{
  req->cb = cb;
  uint32_t blockCount = Bdev_ComputeBlockCount(bd, pkt);

  int iovcnt = Bdev_PrepareIovec(bd, req, pkt, blockCount << bd->blockSizeLog2, true);
  if (unlikely(iovcnt < 0)) {
    req->cb(req, iovcnt);
    return;
  }

  int res = spdk_bdev_readv_blocks(bd->desc, ch, req->iov, iovcnt, blockOffset, blockCount,
                                   Bdev_Complete, req);
  if (unlikely(res != 0)) {
    cb(req, res);
  }
}

void
Bdev_WritePacket(Bdev* bd, struct spdk_io_channel* ch, struct rte_mbuf* pkt, uint64_t blockOffset,
                 BdevRequestCb cb, BdevRequest* req)
{
  req->cb = cb;
  uint32_t blockCount = Bdev_ComputeBlockCount(bd, pkt);
  int iovcnt = Bdev_PrepareIovec(bd, req, pkt, blockCount << bd->blockSizeLog2, false);
  if (unlikely(iovcnt < 0)) {
    req->cb(req, iovcnt);
    return;
  }

  int res = spdk_bdev_writev_blocks(bd->desc, ch, req->iov, iovcnt, blockOffset, blockCount,
                                    Bdev_Complete, req);
  if (unlikely(res != 0)) {
    cb(req, res);
  }
}
