#ifndef NDN_DPDK_NDN_TLV_DECODE_H
#define NDN_DPDK_NDN_TLV_DECODE_H

/// \file

#include "common.h"

/** \brief Parse a TLV-TYPE or TLV-LENGTH number.
 *  \param[out] n the number.
 *  \return number of consumed bytes, or 0 if input is incomplete.
 */
static __rte_always_inline uint32_t
ParseVarNum(const uint8_t* input, uint32_t rem, uint32_t* n)
{
  if (unlikely(rem == 0)) {
    return 0;
  }

  uint8_t firstOctet = *input;
  switch (firstOctet) {
    case 253:
      if (unlikely(rem < 3)) {
        return 0;
      }
      *n = rte_be_to_cpu_16(*(unaligned_uint16_t*)(input + 1));
      return 3;
    case 254:
      if (unlikely(rem < 5)) {
        return 0;
      }
      *n = rte_be_to_cpu_32(*(unaligned_uint32_t*)(input + 1));
      return 5;
    case 255:
      if (unlikely(rem < 9)) {
        return 0;
      }
      *n = rte_be_to_cpu_64(*(unaligned_uint64_t*)(input + 1));
      return 9;
    default:
      *n = firstOctet;
      return 1;
  }
}

/** \brief Parse TLV-TYPE and TLV-LENGTH.
 *  \param[out] type TLV-TYPE number.
 *  \param[out] length TLV-LENGTH number.
 *  \return number of consumed bytes, or 0 if input is incomplete.
 */
static __rte_always_inline uint32_t
ParseTlvTypeLength(const uint8_t* input,
                   uint32_t rem,
                   uint32_t* type,
                   uint32_t* length)
{
  uint32_t sizeofType = ParseVarNum(input, rem, type);
  uint32_t sizeofLength =
    ParseVarNum(input + sizeofType, rem - sizeofLength, length);
  return sizeofType + sizeofLength;
}

#endif // NDN_DPDK_NDN_TLV_DECODE_POS_H
