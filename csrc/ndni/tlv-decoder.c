#include "tlv-decoder.h"

void
TlvDecoder_Truncate(TlvDecoder* d) {
  uint32_t length = d->length;
  NDNDPDK_ASSERT(length > 0);

  if (unlikely(d->m != d->p)) { // d->m is not the first segment
    // move 1 octet from d->m to d->p so that the first segment is non-empty
    NDNDPDK_ASSERT(d->p->data_len >= 1);
    d->p->pkt_len -= (d->p->data_len - 1);
    d->p->data_len = 1;
    d->p->data_off = RTE_PKTMBUF_HEADROOM;
    TlvDecoder_Copy(d, rte_pktmbuf_mtod(d->p, uint8_t*), 1);

    if (unlikely(d->length == 0)) { // 1-octet packet, already in d->p
      d->p->nb_segs -= Mbuf_FreeSegs(&d->p->next, NULL, &d->p->pkt_len);
      goto FINISH;
    }

    // delete segments after d->p and before d->m
    d->p->nb_segs -= Mbuf_FreeSegs(&d->p->next, d->m, &d->p->pkt_len);
  }

  // delete d->m[:d->offset] range
  d->p->pkt_len -= d->offset;
  d->m->data_len -= d->offset;
  d->m->data_off += d->offset;
  d->offset = 0;

  // skip over to end of fragment
  struct rte_mbuf* last = d->m;
  TlvDecoder_Skip_(d, d->length, &last);
  if (d->offset == 0) { // d->m does not contain useful octets
    // delete d->m and segments after d->m
    d->p->nb_segs -= Mbuf_FreeSegs(&last->next, NULL, &d->p->pkt_len);
  } else { // d->m contains useful octets
    // delete d->m[d->offset:] range
    d->p->pkt_len -= (d->m->data_len - d->offset);
    d->m->data_len = d->offset;

    // delete segments after d->m
    d->p->nb_segs -= Mbuf_FreeSegs(&d->m->next, NULL, &d->p->pkt_len);
  }

FINISH:;
  NDNDPDK_ASSERT(d->p->pkt_len == length);
  POISON(d);
}

void
TlvDecoder_Copy_(TlvDecoder* d, uint8_t* output, uint16_t count) {
  for (uint16_t remain = count; remain > 0;) {
    uint16_t here = d->m->data_len - d->offset;
    if (remain < here) {
      rte_memcpy(output, rte_pktmbuf_mtod_offset(d->m, const uint8_t*, d->offset), remain);
      d->offset += remain;
      break;
    }

    rte_memcpy(output, rte_pktmbuf_mtod_offset(d->m, const uint8_t*, d->offset), here);
    output = (uint8_t*)RTE_PTR_ADD(output, here);
    remain -= here;
    d->m = d->m->next;
    d->offset = 0;
  }
  d->length -= count;
}

struct rte_mbuf*
TlvDecoder_Clone(TlvDecoder* d, uint32_t count, struct rte_mempool* indirectMp) {
  NDNDPDK_ASSERT(count <= d->length);
  TlvDecoder d0 = *d;

  uint16_t nSegs = 0;
  for (uint32_t remain = count; remain > 0;) {
    uint32_t here = d0.m->data_len - d0.offset;
    if (likely(remain < here)) {
      d0.offset += remain;
      ++nSegs;
      break;
    }
    if (likely(here > 0)) {
      ++nSegs;
    }
    remain -= here;
    d0.m = d0.m->next;
    d0.offset = 0;
  }

  struct rte_mbuf* segs[RTE_MBUF_MAX_NB_SEGS];
  if (unlikely(rte_pktmbuf_alloc_bulk(indirectMp, segs, nSegs) != 0)) {
    return NULL;
  }

  uint16_t i = 0;
  for (uint32_t remain = count; remain > 0;) {
    struct rte_mbuf* mi = segs[i];

    uint32_t here = d->m->data_len - d->offset;
    if (likely(remain < here)) {
      rte_pktmbuf_attach(mi, d->m);
      rte_pktmbuf_adj(mi, d->offset);
      mi->pkt_len = mi->data_len = remain;
      ++i;

      d->offset += remain;
      break;
    }
    if (likely(here > 0)) {
      rte_pktmbuf_attach(mi, d->m);
      rte_pktmbuf_adj(mi, d->offset);
      ++i;
    }

    remain -= here;
    d->m = d->m->next;
    d->offset = 0;
  }
  d->length -= count;
  NDNDPDK_ASSERT(i == nSegs);

  struct rte_mbuf* pkt = Mbuf_ChainVector(segs, nSegs);
  NDNDPDK_ASSERT(pkt != NULL);
  return pkt;
}

__attribute__((nonnull)) static void
Fragment_Put(TlvDecoder* d, struct rte_mbuf* frame, uint16_t fragSize) {
  uint8_t* room = (uint8_t*)rte_pktmbuf_append(frame, fragSize);
  NDNDPDK_ASSERT(room != NULL);
  TlvDecoder_Copy(d, room, fragSize);
}

void
TlvDecoder_Fragment(TlvDecoder* d, uint32_t count, struct rte_mbuf* frames[], uint32_t* fragIndex,
                    uint32_t fragCount, uint16_t fragSize, uint16_t headroom) {
  NDNDPDK_ASSERT(count <= d->length);

  uint16_t thisFragSize = 0;
  {
    uint16_t targetTail = headroom + fragSize;
    uint16_t currentTail = frames[*fragIndex]->data_off + frames[*fragIndex]->data_len;
    if (currentTail < targetTail) {
      thisFragSize = targetTail - currentTail;
    }
  }

  while (count >= (uint32_t)thisFragSize) {
    if (likely(thisFragSize > 0)) {
      Fragment_Put(d, frames[*fragIndex], thisFragSize);
      count -= thisFragSize;
    }

    ++*fragIndex;
    if (unlikely(*fragIndex >= fragCount)) {
      NDNDPDK_ASSERT(count == 0);
      return;
    }

    frames[*fragIndex]->data_off = headroom;
    thisFragSize = fragSize;
  }

  Fragment_Put(d, frames[*fragIndex], count);
}

__attribute__((nonnull)) static void
Linearize_Delete_(TlvDecoder* d, struct rte_mbuf* c) {
  uint32_t unusedPktLen = UINT32_MAX;
  d->p->nb_segs -= Mbuf_FreeSegs(&c->next, d->m, &unusedPktLen);
  if (likely(d->m != NULL)) {
    d->m->data_len -= d->offset;
    d->m->data_off += d->offset;
  }
  d->offset = 0;
}

__attribute__((nonnull)) static const uint8_t*
Linearize_MoveToFirst_(TlvDecoder* d, uint16_t count) {
  struct rte_mbuf* c = d->m;
  uint16_t co = d->offset;
  if (unlikely(c->data_off + co + count > c->buf_len)) {
    memmove(c->buf_addr, rte_pktmbuf_mtod(c, void*), c->data_len);
    c->data_off = 0;
  }

  uint16_t here = c->data_len - co;
  uint16_t remain = count - here;
  uint8_t* room = rte_pktmbuf_mtod_offset(c, uint8_t*, c->data_len);
  c->data_len += remain;

  d->m = c->next;
  d->offset = 0;
  d->length -= here;
  TlvDecoder_Copy_(d, room, remain);

  Linearize_Delete_(d, c);
  return rte_pktmbuf_mtod_offset(c, const uint8_t*, co);
}

__attribute__((nonnull)) static const uint8_t*
Linearize_CopyToNew_(TlvDecoder* d, uint16_t count) {
  struct rte_mbuf* c = d->m;
  uint16_t co = d->offset;
  NDNDPDK_ASSERT(co != 0); // d->offset==0 belongs to MoveToFirst

  struct rte_mbuf* r = rte_pktmbuf_alloc(d->m->pool);
  if (unlikely(r == NULL)) {
    return NULL;
  }
  r->data_off = 0;
  uint8_t* output = (uint8_t*)rte_pktmbuf_append(r, count);
  NDNDPDK_ASSERT(output != NULL); // dataroom is checked by caller
  TlvDecoder_Copy_(d, output, count);

  r->next = c->next;
  c->next = r;
  c->data_len = co;
  ++d->p->nb_segs;
  Linearize_Delete_(d, r);
  return rte_pktmbuf_mtod(r, const uint8_t*);
}

const uint8_t*
TlvDecoder_Linearize_NonContiguous_(TlvDecoder* d, uint16_t count) {
  NDNDPDK_ASSERT(RTE_MBUF_DIRECT(d->m) && rte_mbuf_refcnt_read(d->m) == 1);

  if (likely(d->offset + count <= d->m->buf_len)) { // result fits in d->m
    return Linearize_MoveToFirst_(d, count);
  }

  if (unlikely(count > d->m->buf_len)) { // insufficient dataroom
    return NULL;
  }

  return Linearize_CopyToNew_(d, count);
}
