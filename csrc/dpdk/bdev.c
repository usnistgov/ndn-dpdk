#include "bdev.h"

void* BdevFillers_ = NULL;

__attribute__((nonnull)) static int
Bdev_PrepareIovec(Bdev* bd, BdevRequest* req, struct rte_mbuf* pkt, uint32_t blockCount,
                  void* filler)
{
  if (unlikely(blockCount == 0 || pkt->nb_segs > BdevMaxMbufSegs)) {
    return -EMSGSIZE;
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

  size_t fillerLen = (blockCount << bd->blockSizeLog2) - pkt->pkt_len;
  if (likely(rte_pktmbuf_tailroom(lastSeg) >= fillerLen)) {
    req->iov[iovcnt - 1].iov_len += fillerLen;
    return iovcnt;
  }

  req->iov[iovcnt].iov_base = filler;
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
  int iovcnt =
    Bdev_PrepareIovec(bd, req, pkt, blockCount, RTE_PTR_ADD(BdevFillers_, BdevFillerLen_));
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
  int iovcnt = Bdev_PrepareIovec(bd, req, pkt, blockCount, RTE_PTR_ADD(BdevFillers_, 0));
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
