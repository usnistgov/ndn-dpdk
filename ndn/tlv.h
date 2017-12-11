#ifndef NDN_TRAFFIC_DPDK_NDN_TLV_H
#define NDN_TRAFFIC_DPDK_NDN_TLV_H

#include "common.h"

#define VARNUM_MAXLENGTH 9

size_t __EncodeVarNum_MultiOctet(uint64_t n, uint8_t* output);

// Encode a TLV-TYPE or TLV-LENGTH number.
// n: the number.
// output [output]: buffer, must have at least VARNUM_MAXLENGTH octets.
// Return length of encoded number.
static inline size_t
EncodeVarNum(uint64_t n, uint8_t* output)
{
  if (unlikely(n >= 253)) {
    return __EncodeVarNum_MultiOctet(n, output);
  }

  output[0] = (uint8_t)n;
  return 1;
}

NdnError __DecodeVarNum_MultiOctet(const uint8_t* input, size_t inputLen,
                                   uint64_t* n, size_t* len);

// Decode a TLV-TYPE or TLV-LENGTH number.
// input, inputLen: input buffer and its length.
// n [output]: the number.
// len [output]: length of encoded number.
static inline NdnError
DecodeVarNum(const uint8_t* input, size_t inputLen, uint64_t* n, size_t* len)
{
  if (unlikely(inputLen < 1)) {
    return NdnError_BufferTooSmall;
  }

  if (unlikely(input[0] >= 253)) {
    return __DecodeVarNum_MultiOctet(input, inputLen, n, len);
  }

  *len = 1;
  *n = input[0];
  return NdnError_OK;
}

#endif // NDN_TRAFFIC_DPDK_NDN_TLV_H