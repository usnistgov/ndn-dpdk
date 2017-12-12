#include "mbuf.h"

// Same as MbucLoc_Diff but only consider one direction: advance a to reach b.
static inline bool
MbufLoc_Diff_OneSided(const MbufLoc* a, const MbufLoc* b, ptrdiff_t* dist)
{
  *dist = 0;
  struct rte_mbuf* am = a->m;
  uint16_t aOff = a->off;
  struct rte_mbuf* bm = b->m;
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
  if (MbufLoc_Diff_OneSided(a, b, &dist)) {
    return dist;
  }
  if (MbufLoc_Diff_OneSided(b, a, &dist)) {
    return -dist;
  }
  assert(false);
}

uint32_t
__MbufLoc_Read_MultiSeg(MbufLoc* ml, void* output, uint32_t n)
{
  uint8_t* data = rte_pktmbuf_mtod_offset(ml->m, uint8_t*, ml->off);
  rte_prefetch0(data);

  uint32_t nRead = 0;
  uint32_t last = ml->off + n;
  while (last >= ml->m->data_len) {
    uint16_t nCopy = ml->m->data_len - ml->off;
    rte_memcpy(output, data, nCopy);
    last -= ml->m->data_len;
    ml->m = ml->m->next;
    nRead += nCopy;

    if (unlikely(ml->m == NULL)) {
      return nRead;
    }
    data = rte_pktmbuf_mtod(ml->m, uint8_t*);
    rte_prefetch0(data);

    ml->off = 0;
    output += nCopy;
  }

  rte_memcpy(output, data, last);
  nRead += last;
  ml->off = (uint16_t)last;
  return nRead;
}