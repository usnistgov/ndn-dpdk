#include "mbuf-loc.h"

// Same as MbucLoc_Diff but only consider one direction: advance a to reach b.
static inline bool
__MbufLoc_Diff_OneSided(const MbufLoc* a, const MbufLoc* b, ptrdiff_t* dist)
{
  *dist = 0;
  const struct rte_mbuf* am = a->m;
  uint16_t aOff = a->off;
  const struct rte_mbuf* bm = b->m;
  while (am != NULL) {
    if (am == bm) {
      *dist += b->off - aOff;
      return true;
    }
    *dist += am->data_len - aOff;
    am = am->next;
    aOff = 0;
  }
  return false;
}

ptrdiff_t
MbufLoc_Diff(const MbufLoc* a, const MbufLoc* b)
{
  assert(!MbufLoc_IsEnd(a) && !MbufLoc_IsEnd(b));

  ptrdiff_t dist = 0;
  if (__MbufLoc_Diff_OneSided(a, b, &dist)) {
    return dist;
  }
  if (__MbufLoc_Diff_OneSided(b, a, &dist)) {
    return -dist;
  }
  assert(false);
}

void
__MbufLoc_MakeIndirectCb(void* arg, const struct rte_mbuf* m, uint16_t off,
                         uint16_t len)
{
  __MbufLoc_MakeIndirectCtx* ctx = (__MbufLoc_MakeIndirectCtx*)arg;
  if (unlikely(ctx->mp == NULL)) {
    return;
  }

  struct rte_mbuf* mi = rte_pktmbuf_alloc(ctx->mp);
  if (unlikely(mi == NULL)) {
    ctx->mp = NULL;
    return;
  }

  rte_pktmbuf_attach(mi, (struct rte_mbuf*)m);
  if (ctx->head == NULL) {
    ctx->head = mi;
    mi->nb_segs = 0;
  }

  rte_pktmbuf_adj(mi, off);
  rte_pktmbuf_trim(mi, mi->data_len - len);
  mi->pkt_len = 0;
  ctx->head->pkt_len += mi->data_len;
  ++ctx->head->nb_segs;

  if (ctx->tail != NULL) {
    ctx->tail->next = mi;
  }
  ctx->tail = mi;
}

void
__MbufLoc_ReadCb(void* arg, const struct rte_mbuf* m, uint16_t off,
                 uint16_t len)
{
  uint8_t** output = (uint8_t**)arg;
  uint8_t* input = rte_pktmbuf_mtod_offset(m, uint8_t*, off);
  rte_memcpy(*output, input, len);
  *output += len;
}

// Find previous segment.
static inline struct rte_mbuf*
__MbufLoc_FindPrev(const struct rte_mbuf* m, struct rte_mbuf* pkt)
{
  assert(m != pkt);
  struct rte_mbuf* prev = pkt;
  while (prev->next != m) {
    prev = prev->next;
  }
  return prev;
}

void
MbufLoc_Delete(MbufLoc* ml, uint32_t n, struct rte_mbuf* pkt,
               struct rte_mbuf* prev)
{
  if (unlikely(n == 0)) {
    return;
  }
  assert(!MbufLoc_IsEnd(ml));
  assert(prev == NULL || prev->next == ml->m);

  uint32_t oldPktLen = pkt->pkt_len;
  struct rte_mbuf* firstM = (struct rte_mbuf*)ml->m;

  if (ml->off + n <= firstM->data_len) { // is the range inside firstM?
    if (ml->off == 0) {
      // delete first n octets
      firstM->data_off += n;
    } else {
      // move [ml->off+n,end) to ml->off
      uint16_t nMoving = firstM->data_len - ml->off - n;
      if (likely(nMoving > 0)) {
        uint8_t* dst = rte_pktmbuf_mtod_offset(firstM, uint8_t*, ml->off);
        const uint8_t* src =
          rte_pktmbuf_mtod_offset(firstM, uint8_t*, ml->off + n);
        memmove(dst, src, nMoving);
      }
    }
    firstM->data_len -= n;
    pkt->pkt_len -= n;
  } else {
    // in firstM, delete [ml->off,end)
    uint16_t nTrim = firstM->data_len - ml->off;
    firstM->data_len -= nTrim;
    pkt->pkt_len -= nTrim;
    uint32_t remaining = n - nTrim;

    // delete remaining octets from subsequent segments
    while (remaining > 0) {
      struct rte_mbuf* seg = firstM->next;
      assert(seg != NULL);
      bool isEmptying = seg->data_len <= remaining;
      if (isEmptying) {
        // segment becomes empty
        pkt->pkt_len -= seg->data_len;
        remaining -= seg->data_len;
        firstM->next = seg->next;
        --pkt->nb_segs;
        rte_pktmbuf_free_seg(seg);
      } else {
        // delete first n octets from the segment
        seg->data_off += remaining;
        seg->data_len -= remaining;
        pkt->pkt_len -= remaining;
        remaining = 0;
      }
    }
  }

  // free firstM if it is empty, unless it is the first segment
  if (firstM != pkt && firstM->data_len == 0) {
    if (prev == NULL) {
      prev = __MbufLoc_FindPrev(firstM, pkt);
    } else {
      assert(prev->next == firstM);
    }
    prev->next = firstM->next;
    --pkt->nb_segs;
    rte_pktmbuf_free_seg(firstM);
  }

  assert((rte_mbuf_sanity_check(pkt, 1), true));
  assert(pkt->pkt_len + n == oldPktLen);
}
