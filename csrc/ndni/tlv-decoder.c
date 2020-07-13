#include "tlv-decoder.h"

void
TlvDecoder_Read_NonContiguous_(TlvDecoder* d, uint8_t* output, uint16_t count)
{
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
TlvDecoder_Clone(TlvDecoder* d, uint32_t count, struct rte_mempool* indirectMp,
                 struct rte_mbuf** lastseg)
{
  assert(count <= d->length);
  TlvDecoder d0 = *d;

  unsigned nSegs = 0;
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

  unsigned i = 0;
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
  assert(i == nSegs);

  struct rte_mbuf* head = segs[0];
  for (i = 1; i < nSegs; ++i) {
    struct rte_mbuf* mi = segs[i];
    segs[i - 1]->next = mi;
    head->pkt_len += mi->data_len;
  }
  if (lastseg != NULL) {
    *lastseg = segs[nSegs - 1];
  }
  return head;
}

static void
TlvDecoder_Linearize_Delete_(TlvDecoder* d, struct rte_mbuf* c)
{
  for (struct rte_mbuf* seg = c->next; seg != d->m;) {
    struct rte_mbuf* next = seg->next;
    rte_pktmbuf_free_seg(seg);
    --d->p->nb_segs;
    seg = next;
  }
  c->next = d->m;
  if (likely(d->m != NULL)) {
    d->m->data_len -= d->offset;
    d->m->data_off += d->offset;
  }
  d->offset = 0;
}

static const uint8_t*
TlvDecoder_Linearize_MoveToFirst_(TlvDecoder* d, uint16_t count)
{
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
  TlvDecoder_Read_NonContiguous_(d, room, remain);

  TlvDecoder_Linearize_Delete_(d, c);
  return rte_pktmbuf_mtod_offset(c, const uint8_t*, co);
}

static const uint8_t*
TlvDecoder_Linearize_CopyToNew_(TlvDecoder* d, uint16_t count)
{
  struct rte_mbuf* c = d->m;
  uint16_t co = d->offset;
  assert(co != 0); // d->offset==0 belongs to MoveToFirst

  struct rte_mbuf* r = rte_pktmbuf_alloc(d->m->pool);
  if (unlikely(r == NULL)) {
    return NULL;
  }
  r->data_off = 0;
  uint8_t* output = (uint8_t*)rte_pktmbuf_append(r, count);
  assert(output != NULL); // dataroom is checked by caller
  TlvDecoder_Read_NonContiguous_(d, output, count);

  r->next = c->next;
  c->next = r;
  c->data_len = co;
  ++d->p->nb_segs;
  TlvDecoder_Linearize_Delete_(d, r);
  return rte_pktmbuf_mtod(r, const uint8_t*);
}

const uint8_t*
TlvDecoder_Linearize_NonContiguous_(TlvDecoder* d, uint16_t count)
{
  assert(RTE_MBUF_DIRECT(d->m) && rte_mbuf_refcnt_read(d->m) == 1);

  if (likely(d->offset + count <= d->m->buf_len)) { // result fits in d->m
    return TlvDecoder_Linearize_MoveToFirst_(d, count);
  }

  if (unlikely(count > d->m->buf_len)) { // insufficient dataroom
    return NULL;
  }

  return TlvDecoder_Linearize_CopyToNew_(d, count);
}
