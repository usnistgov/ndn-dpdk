#include "tlv-decoder.h"
#include <rte_byteorder.h>

__rte_noinline NdnError
__DecodeVarNum_MultiOctet(MbufLoc* ml, uint8_t firstOctet, uint64_t* n,
                          size_t* len)
{
  if (unlikely(MbufLoc_IsEnd(ml))) {
    return NdnError_Incomplete;
  }

  switch (firstOctet) {
    case 253: {
      rte_be16_t v;
      bool ok = MbufLoc_ReadU16(ml, &v);
      if (unlikely(!ok)) {
        return NdnError_Incomplete;
      }
      *len = 3;
      *n = rte_be_to_cpu_16(v);
      break;
    }
    case 254: {
      rte_be32_t v;
      bool ok = MbufLoc_ReadU32(ml, &v);
      if (unlikely(!ok)) {
        return NdnError_Incomplete;
      }
      *len = 5;
      *n = rte_be_to_cpu_32(v);
      break;
    }
    case 255: {
      rte_be64_t v;
      bool ok = MbufLoc_ReadU64(ml, &v);
      if (unlikely(!ok)) {
        return NdnError_Incomplete;
      }
      *len = 9;
      *n = rte_be_to_cpu_64(v);
      break;
    }
    default:
      assert(false);
  }
  return NdnError_OK;
}