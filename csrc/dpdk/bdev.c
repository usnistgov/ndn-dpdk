#include "bdev.h"

static const size_t SPDK_BDEV_FILLER_LEN = 65536;
static const size_t SPDK_BDEV_FILLER_MAX = 16;
static void* SpdkBdev_Filler = 0;

void
SpdkBdev_InitFiller()
{
  SpdkBdev_Filler = rte_malloc("SpdkBdevFiller", SPDK_BDEV_FILLER_LEN, 0);
  NDNDPDK_ASSERT(SpdkBdev_Filler != NULL);
}

static int
SpdkBdev_MakeIovec(struct rte_mbuf* pkt, uint64_t minLen,
                   struct iovec iov[SPDK_BDEV_MAX_MBUF_SEGS + SPDK_BDEV_FILLER_MAX])
{
  if (pkt->nb_segs > SPDK_BDEV_MAX_MBUF_SEGS) {
    return -EMSGSIZE;
  }

  size_t i = 0;
  for (struct rte_mbuf* seg = pkt; seg != NULL; seg = seg->next) {
    if (unlikely(seg->data_len == 0)) {
      continue;
    }
    iov[i].iov_base = rte_pktmbuf_mtod(seg, void*);
    iov[i].iov_len = seg->data_len;
    ++i;
  }

  // XXX filler is unnecessary for AIO and NVMe
  for (uint64_t len = pkt->pkt_len; unlikely(len < minLen); len += SPDK_BDEV_FILLER_LEN) {
    if (unlikely(i >= SPDK_BDEV_MAX_MBUF_SEGS + SPDK_BDEV_FILLER_MAX)) {
      return -EMSGSIZE;
    }
    NDNDPDK_ASSERT(SpdkBdev_Filler != 0);
    iov[i].iov_base = (void*)SpdkBdev_Filler;
    iov[i].iov_len = SPDK_BDEV_FILLER_LEN;
    ++i;
  }

  return i;
}

int
SpdkBdev_ReadPacket(struct spdk_bdev_desc* desc, struct spdk_io_channel* ch, struct rte_mbuf* pkt,
                    uint64_t blockOffset, uint64_t blockCount, uint32_t blockSize,
                    spdk_bdev_io_completion_cb cb, void* ctx)
{
  struct iovec iov[SPDK_BDEV_MAX_MBUF_SEGS + SPDK_BDEV_FILLER_MAX];
  int iovcnt = SpdkBdev_MakeIovec(pkt, blockCount * blockSize, iov);
  if (unlikely(iovcnt < 0)) {
    return iovcnt;
  }
  return spdk_bdev_readv_blocks(desc, ch, iov, iovcnt, blockOffset, blockCount, cb, ctx);
}

int
SpdkBdev_WritePacket(struct spdk_bdev_desc* desc, struct spdk_io_channel* ch, struct rte_mbuf* pkt,
                     uint64_t blockOffset, uint64_t blockCount, uint32_t blockSize,
                     spdk_bdev_io_completion_cb cb, void* ctx)
{
  struct iovec iov[SPDK_BDEV_MAX_MBUF_SEGS + SPDK_BDEV_FILLER_MAX];
  int iovcnt = SpdkBdev_MakeIovec(pkt, blockCount * blockSize, iov);
  if (unlikely(iovcnt < 0)) {
    return iovcnt;
  }
  return spdk_bdev_writev_blocks(desc, ch, iov, iovcnt, blockOffset, blockCount, cb, ctx);
}
