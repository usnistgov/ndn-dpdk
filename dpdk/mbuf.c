#include "mbuf.h"

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