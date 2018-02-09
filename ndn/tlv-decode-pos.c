#include "tlv-decode-pos.h"

NdnError
__DecodeVarNum_32or64(TlvDecodePos* d, uint8_t firstOctet, uint64_t* n)
{
  if (unlikely(MbufLoc_IsEnd(d))) {
    return NdnError_Incomplete;
  }

  switch (firstOctet) {
    case 254: {
      rte_be32_t v;
      bool ok = MbufLoc_ReadU32(d, &v);
      if (unlikely(!ok)) {
        return NdnError_Incomplete;
      }
      *n = rte_be_to_cpu_32(v);
      break;
    }
    case 255: {
      rte_be64_t v;
      bool ok = MbufLoc_ReadU64(d, &v);
      if (unlikely(!ok)) {
        return NdnError_Incomplete;
      }
      *n = rte_be_to_cpu_64(v);
      break;
    }
    default:
      assert(false);
  }
  return NdnError_OK;
}
