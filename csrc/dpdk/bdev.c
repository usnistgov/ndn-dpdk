#include "bdev.h"

enum
{
  FILLER_LEN = 65536,
  FILLER_MAX = 16,
  NIOV_MAX = SPDK_BDEV_MAX_MBUF_SEGS + FILLER_MAX,
};
static void* Filler = NULL;

void
SpdkBdev_InitFiller()
{
  Filler = rte_malloc("SpdkBdevFiller", FILLER_LEN, 0);
  NDNDPDK_ASSERT(Filler != NULL);
}

__attribute__((nonnull)) static int
SpdkBdev_MakeIovec(struct rte_mbuf* pkt, size_t totalLen, struct iovec iov[NIOV_MAX])
{
  if (pkt->nb_segs > SPDK_BDEV_MAX_MBUF_SEGS) {
    return -EMSGSIZE;
  }

  size_t nIov = 0;
  for (struct rte_mbuf* seg = pkt; seg != NULL; seg = seg->next) {
    if (unlikely(seg->data_len == 0)) {
      continue;
    }
    iov[nIov].iov_base = rte_pktmbuf_mtod(seg, void*);
    iov[nIov].iov_len = seg->data_len;
    ++nIov;
  }

  // Malloc driver expects SUM(iov[].iov_len) == blockCount*blockSize
  // XXX filler is unnecessary for AIO and NVMe drivers
  size_t fillersLen = totalLen - pkt->pkt_len;
  size_t nFillers = fillersLen / FILLER_LEN;
  size_t firstFiller = fillersLen % FILLER_LEN;
  if (firstFiller != 0) {
    iov[nIov].iov_base = Filler;
    iov[nIov].iov_len = firstFiller;
    ++nIov;
  }
  if (unlikely(nIov + nFillers >= NIOV_MAX)) {
    return -EMSGSIZE;
  }
  for (size_t i = 0; i < nFillers; ++i) {
    iov[nIov].iov_base = Filler;
    iov[nIov].iov_len = FILLER_LEN;
    ++nIov;
  }
  return nIov;
}

int
SpdkBdev_ReadPacket(struct spdk_bdev_desc* desc, struct spdk_io_channel* ch, struct rte_mbuf* pkt,
                    uint64_t blockOffset, uint64_t blockCount, uint32_t blockSize,
                    spdk_bdev_io_completion_cb cb, uintptr_t ctx)
{
  struct iovec iov[NIOV_MAX];
  int iovcnt = SpdkBdev_MakeIovec(pkt, blockCount * blockSize, iov);
  if (unlikely(iovcnt < 0)) {
    return iovcnt;
  }
  return spdk_bdev_readv_blocks(desc, ch, iov, iovcnt, blockOffset, blockCount, cb, (void*)ctx);
}

int
SpdkBdev_WritePacket(struct spdk_bdev_desc* desc, struct spdk_io_channel* ch, struct rte_mbuf* pkt,
                     uint64_t blockOffset, uint64_t blockCount, uint32_t blockSize,
                     spdk_bdev_io_completion_cb cb, uintptr_t ctx)
{
  struct iovec iov[NIOV_MAX];
  int iovcnt = SpdkBdev_MakeIovec(pkt, blockCount * blockSize, iov);
  if (unlikely(iovcnt < 0)) {
    return iovcnt;
  }
  return spdk_bdev_writev_blocks(desc, ch, iov, iovcnt, blockOffset, blockCount, cb, (void*)ctx);
}
