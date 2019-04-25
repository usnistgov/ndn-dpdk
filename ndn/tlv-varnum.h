#ifndef NDN_DPDK_NDN_TLV_VARNUM_H
#define NDN_DPDK_NDN_TLV_VARNUM_H

/// \file

#include "common.h"

/** \brief Compute size of a TLV-TYPE or TLV-LENGTH number.
 */
static int
SizeofVarNum(uint64_t n)
{
  return n <= UINT16_MAX ? (n < 253 ? 1 : 3) : (n <= UINT32_MAX ? 5 : 9);
}

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

uint8_t*
__EncodeVarNum_32or64(uint8_t* room, uint64_t n);

/** \brief Encode a TLV-TYPE or TLV-LENGTH number.
 *  \param[out] room output buffer, must have \c SizeofVarNum(n) octets
 *  \param n the number
 *  \return room + SizeofVarNum(n)
 */
static uint8_t*
EncodeVarNum(uint8_t* room, uint64_t n)
{
  if (unlikely(n > UINT16_MAX)) {
    return __EncodeVarNum_32or64(room, n);
  }

  if (n < 253) {
    room[0] = (uint8_t)n;
    return room + 1;
  } else {
    room[0] = 253;
    room[1] = (uint8_t)(n >> 8);
    room[2] = (uint8_t)n;
    return room + 3;
  }
}

#endif // NDN_DPDK_NDN_TLV_VARNUM_H
