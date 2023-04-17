#include "mbuf.h"

static_assert(sizeof(rte_mbuf_timestamp_t) == sizeof(TscTime), "");

int Mbuf_Timestamp_DynFieldOffset_ = -1;

bool
Mbuf_RegisterDynFields()
{
  int res = rte_mbuf_dyn_rx_timestamp_register(&Mbuf_Timestamp_DynFieldOffset_, NULL);
  return res == 0;
}

int
Mbuf_AsIovec(struct rte_mbuf* m, struct iovec* iov, uint32_t offset, uint32_t length)
{
  int iovcnt = 0;
  for (struct rte_mbuf* seg = m; seg != NULL && length > 0; seg = seg->next) {
    uint32_t skipLen = RTE_MIN(offset, (uint32_t)seg->data_len);
    offset -= skipLen;
    uint32_t acceptLen = RTE_MIN(length, (uint32_t)seg->data_len - skipLen);
    length -= acceptLen;
    if (acceptLen == 0) {
      continue;
    }
    iov[iovcnt++] = (struct iovec){
      .iov_base = rte_pktmbuf_mtod_offset(seg, void*, skipLen),
      .iov_len = acceptLen,
    };
  }
  return iovcnt;
}

struct rte_mbuf*
Mbuf_AllocRoom(struct rte_mempool* mp, struct iovec* iov, int* iovcnt, uint16_t firstHeadroom,
               uint16_t firstDataLen, uint16_t eachHeadroom, uint16_t eachDataLen, uint32_t pktLen)
{
  uint16_t dataroom = rte_pktmbuf_data_room_size(mp);
  if (unlikely(firstHeadroom + firstDataLen > dataroom || eachHeadroom + eachDataLen > dataroom)) {
    rte_errno = E2BIG;
    return NULL;
  }
  if (firstDataLen == 0) {
    firstDataLen = dataroom - firstHeadroom;
  }
  if (eachDataLen == 0) {
    eachDataLen = dataroom - eachHeadroom;
  }

  int nSegs = 1;
  if (pktLen > firstDataLen) {
    nSegs += SPDK_CEIL_DIV(pktLen - firstDataLen, eachDataLen);
  }
  struct rte_mbuf* segs[64];
  if (unlikely(nSegs > RTE_MIN(*iovcnt, (int)RTE_DIM(segs)))) {
    rte_errno = EFBIG;
    return NULL;
  }

  int res = rte_pktmbuf_alloc_bulk(mp, segs, nSegs);
  if (unlikely(res != 0)) {
    rte_errno = res;
    return NULL;
  }

  uint16_t thisHeadroom = firstHeadroom;
  uint32_t thisDataLen = firstDataLen;
  uint32_t sumDataLen = 0;
  for (int i = 0; i < nSegs; ++i) {
    struct rte_mbuf* seg = segs[i];
    seg->data_off = thisHeadroom;
    thisDataLen = RTE_MIN(thisDataLen, pktLen - sumDataLen);
    iov[i] = (struct iovec){
      .iov_base = rte_pktmbuf_append(seg, thisDataLen),
      .iov_len = thisDataLen,
    };
    sumDataLen += thisDataLen;
    thisHeadroom = eachHeadroom;
    thisDataLen = eachDataLen;
  }
  *iovcnt = nSegs;
  rte_errno = 0;
  return Mbuf_ChainVector(segs, nSegs);
}

void
Mbuf_RemainingIovec(struct spdk_iov_xfer ix, struct iovec* iov, int* iovcnt)
{
  while (
    unlikely(ix.cur_iov_idx < ix.iovcnt && ix.iovs[ix.cur_iov_idx].iov_len == ix.cur_iov_offset)) {
    ++ix.cur_iov_idx;
    ix.cur_iov_offset = 0;
  }
  if (unlikely(ix.cur_iov_idx >= ix.iovcnt)) {
    *iovcnt = 0;
    return;
  }

  *iovcnt = ix.iovcnt - ix.cur_iov_idx;
  memmove(iov, &ix.iovs[ix.cur_iov_idx], (*iovcnt) * sizeof(iov[0]));
  iov[0].iov_len -= ix.cur_iov_offset;
  iov[0].iov_base = RTE_PTR_ADD(iov[0].iov_base, ix.cur_iov_offset);
}
