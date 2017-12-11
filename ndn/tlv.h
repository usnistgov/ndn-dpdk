#ifndef NDN_TRAFFIC_DPDK_NDN_TLV_H
#define NDN_TRAFFIC_DPDK_NDN_TLV_H

#include "common.h"
#include <rte_byteorder.h>

// TLV-TYPE and TLV-LENGTH variable size number encoding

// Required buffer size for varnum_* functions.
#define VARNUM_BUFLEN 9

// Encode n into output.
// n: the number.
// output [output]: buffer, must have at least VARNUM_BUFLEN octets.
// Return length of encoded number.
static inline size_t
EncodeVarNum(uint64_t n, uint8_t* output)
{
  if (unlikely(n > UINT32_MAX)) {
    output[0] = 255;
    static_assert(sizeof(uint64_t) == 8, "");
    *(rte_be64_t*)(output + 1) = rte_cpu_to_be_64(n);
    return 9;
  }

  if (unlikely(n > UINT16_MAX)) {
    output[0] = 254;
    static_assert(sizeof(uint32_t) == 4, "");
    *(rte_be32_t*)(output + 1) = rte_cpu_to_be_32((uint32_t)n);
    return 5;
  }

  if (unlikely(n >= 253)) {
    output[0] = 253;
    static_assert(sizeof(uint16_t) == 2, "");
    *(rte_be16_t*)(output + 1) = rte_cpu_to_be_16((uint16_t)n);
    return 3;
  }

  output[0] = (uint8_t)n;
  return 1;
}

// Decode input into n.
// input: buffer, must have at least VARNUM_BUFLEN octets.
// n [output]: the number.
// Return length of encoded number.
static inline size_t
DecodeVarNum(const uint8_t* input, uint64_t* n)
{
  if (unlikely(input[0] == 255)) {
    *n = rte_be_to_cpu_64(*(const rte_be64_t*)(input + 1));
    return 9;
  }

  if (unlikely(input[0] == 254)) {
    *n = rte_be_to_cpu_32(*(const rte_be32_t*)(input + 1));
    return 5;
  }

  if (unlikely(input[0] == 253)) {
    *n = rte_be_to_cpu_16(*(const rte_be16_t*)(input + 1));
    return 3;
  }

  *n = input[0];
  return 1;
}

#endif // NDN_TRAFFIC_DPDK_NDN_TLV_H