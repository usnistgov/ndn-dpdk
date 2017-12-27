#include "tlv-encoder.h"

void
__EncodeVarNum_32or64(uint8_t* room, uint64_t n)
{
  assert(n > UINT16_MAX);
  if (n <= UINT32_MAX) {
    *room++ = 254;
    rte_be32_t v = rte_cpu_to_be_32((uint32_t)n);
    rte_memcpy(room, &v, 4);
  } else {
    *room++ = 255;
    rte_be64_t v = rte_cpu_to_be_64(n);
    rte_memcpy(room, &v, 8);
  }
}