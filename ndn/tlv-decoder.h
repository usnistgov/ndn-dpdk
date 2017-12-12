#ifndef NDN_TRAFFIC_DPDK_NDN_TLV_DECODER_H
#define NDN_TRAFFIC_DPDK_NDN_TLV_DECODER_H

#include "common.h"

NdnError __DecodeVarNum_MultiOctet(MbufLoc* ml, uint8_t firstOctet, uint64_t* n,
                                   size_t* len);

// Decode a TLV-TYPE or TLV-LENGTH number.
// ml: input mbuf and position.
// n [output]: the number.
// len [output]: length of encoded number.
// Return OK if successful, input is advanced past the end of VarNum.
// Return Incomplete if reaching end, input is not preserved.
static inline NdnError
DecodeVarNum(MbufLoc* ml, uint64_t* n, size_t* len)
{
  if (unlikely(MbufLoc_IsEnd(ml))) {
    return NdnError_Incomplete;
  }

  uint8_t firstOctet;
  bool ok = MbufLoc_ReadU8(ml, &firstOctet);
  if (unlikely(!ok)) {
    return NdnError_Incomplete;
  }

  if (unlikely(firstOctet >= 253)) {
    return __DecodeVarNum_MultiOctet(ml, firstOctet, n, len);
  }

  *len = 1;
  *n = firstOctet;
  return NdnError_OK;
}

#endif // NDN_TRAFFIC_DPDK_NDN_TLV_DECODER_H