#include "mbuf-loc.h"

// Determine the distance advancing a to reach b.
static bool
__MbufLoc_Diff_Forward(const MbufLoc* a, const MbufLoc* b, ptrdiff_t* dist)
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

// Determine the distance advancing ml to reach the end.
static ptrdiff_t
__MbufLoc_Diff_ToEnd(const MbufLoc* ml)
{
  ptrdiff_t dist = 0;
  const struct rte_mbuf* m = ml->m;
  uint16_t off = ml->off;
  while (m != NULL) {
    dist += m->data_len - off;
    m = m->next;
    off = 0;
  }
  return dist;
}

ptrdiff_t
MbufLoc_Diff(const MbufLoc* a, const MbufLoc* b)
{
  if (b->m == NULL) {
    return __MbufLoc_Diff_ToEnd(a);
  }
  if (a->m == NULL) {
    return -__MbufLoc_Diff_ToEnd(b);
  }

  ptrdiff_t dist = 0;
  if (__MbufLoc_Diff_Forward(a, b, &dist)) {
    return dist;
  }
  if (__MbufLoc_Diff_Forward(b, a, &dist)) {
    return -dist;
  }
  assert(false);
  return 0;
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
  rte_pktmbuf_adj(mi, off);
  rte_pktmbuf_trim(mi, mi->data_len - len);
  if (ctx->head == NULL) {
    ctx->head = mi;
    mi->nb_segs = 1;
  } else {
    ++ctx->head->nb_segs;
  }
  mi->pkt_len = 0;
  ctx->head->pkt_len += mi->data_len;

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
static struct rte_mbuf*
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

  // 'advance' ml by zero, so that it points to valid buffer
  if (ml->off == firstM->data_len) {
    while (ml->m != NULL && ml->m->data_len == ml->off) {
      ml->m = ml->m->next;
      ml->off = 0;
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

uint8_t*
__MbufLoc_Linearize(MbufLoc* first, MbufLoc* last, struct rte_mbuf* pkt,
                    struct rte_mempool* mp)
{
  struct rte_mbuf* firstM = (struct rte_mbuf*)first->m;
  assert(firstM != last->m); // simple case handled by MbufLoc_Linearize

  ptrdiff_t distance = MbufLoc_Diff(first, last);
  if (unlikely(distance == 0)) {
    return rte_pktmbuf_mtod_offset(firstM, uint8_t*, first->off);
  }
  assert(distance > 0);
  uint32_t n = distance;
  assert(n <= first->rem);

  uint32_t oldPktLen = pkt->pkt_len;

  // how many octets are in firstM?
  uint16_t nInFirst = firstM->data_len - first->off;
  // how many octets need to be copied to the end of firstM?
  uint32_t nCopyingToFirst = n - nInFirst;
  // do they fit in tailroom of firstM?
  bool canAppendToFirst = rte_pktmbuf_tailroom(firstM) >= nCopyingToFirst;

  if (canAppendToFirst) {
    MbufLoc ml;
    ml.m = firstM->next;
    ml.off = 0;
    ml.rem = first->rem - nInFirst;
    MbufLoc_Copy(last, &ml);

    // append to firstM
    uint8_t* dst = rte_pktmbuf_mtod_offset(firstM, uint8_t*, firstM->data_len);
    uint32_t nCopied = MbufLoc_ReadTo(&ml, dst, nCopyingToFirst);
    assert(nCopied == nCopyingToFirst);
    firstM->data_len += nCopied;
    pkt->pkt_len += nCopied;

    // delete copied range
    MbufLoc_Delete(last, nCopied, pkt, firstM);
  } else {
    // allocate linear mbuf
    if (unlikely(rte_pktmbuf_data_room_size(mp) < n)) {
      rte_errno = EMSGSIZE;
      return NULL;
    }
    struct rte_mbuf* linearM = rte_pktmbuf_alloc(mp);
    if (unlikely(linearM == NULL)) {
      rte_errno = ENOMEM;
      return NULL;
    }
    linearM->data_off = 0;
    assert(rte_pktmbuf_tailroom(linearM) >= n);

    // copy to linearM
    MbufLoc ml;
    MbufLoc_Copy(&ml, first);
    uint8_t* dst = rte_pktmbuf_mtod(linearM, uint8_t*);
    uint32_t nCopied = MbufLoc_ReadTo(&ml, dst, n);
    assert(nCopied == n);
    linearM->data_len = n;

    // will MbufLoc_Delete free firstM?
    bool isFreeingFirst = first->off == 0 && firstM != pkt;
    struct rte_mbuf* prev = NULL;
    if (isFreeingFirst) {
      prev = __MbufLoc_FindPrev(firstM, pkt);
    }

    // delete copied range
    MbufLoc_Copy(last, first);
    MbufLoc_Delete(last, n, pkt, prev);
    assert(last->m == NULL || last->off == 0);

    // insert linearM
    if (!isFreeingFirst) {
      prev = firstM;
    }
    assert(prev->next == last->m);
    ++pkt->nb_segs;
    pkt->pkt_len += n;
    prev->next = linearM;
    linearM->next = (struct rte_mbuf*)last->m;

    // point first to linearM
    first->m = linearM;
    first->off = 0;
  }

  assert(pkt->pkt_len == oldPktLen);
  assert(MbufLoc_Diff(first, last) == distance);

  return rte_pktmbuf_mtod_offset(first->m, uint8_t*, first->off);
}
