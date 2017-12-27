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
