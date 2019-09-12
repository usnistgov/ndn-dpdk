#ifndef NDN_DPDK_NDN_TLV_VARNUM_H
#define NDN_DPDK_NDN_TLV_VARNUM_H

/// \file

#include "common.h"

/** \brief Compute size of a TLV-TYPE or TLV-LENGTH number.
 */
static __rte_always_inline int
SizeofVarNum(uint32_t n)
{
  if (n < 253) {
    return 1;
  }
  if (n <= UINT16_MAX) {
    return 3;
  }
  return 5;
}

/** \brief Parse a TLV-TYPE or TLV-LENGTH number.
 *  \param[out] n the number.
 *  \return number of consumed bytes, or negative NdnError.
 */
static __rte_always_inline int
DecodeVarNum(const uint8_t* input, uint32_t rem, uint32_t* n)
{
  if (unlikely(rem == 0)) {
    return -NdnError_Incomplete;
  }

  uint8_t firstOctet = *input;
  switch (firstOctet) {
    case 253:
      if (unlikely(rem < 3)) {
        return -NdnError_Incomplete;
      }
      *n = rte_be_to_cpu_16(*(unaligned_uint16_t*)(input + 1));
      return 3;
    case 254:
      if (unlikely(rem < 5)) {
        return -NdnError_Incomplete;
      }
      *n = rte_be_to_cpu_32(*(unaligned_uint32_t*)(input + 1));
      return 5;
    case 255:
      return -NdnError_LengthOverflow;
    default:
      *n = firstOctet;
      return 1;
  }
}

/** \brief Decode a TLV-TYPE or TLV-LENGTH number.
 *  \param[out] n the number.
 */
static NdnError
MbufLoc_ReadVarNum(MbufLoc* ml, uint32_t* n)
{
  if (unlikely(MbufLoc_IsEnd(ml))) {
    return NdnError_Incomplete;
  }

  const uint8_t* src = rte_pktmbuf_mtod_offset(ml->m, uint8_t*, ml->off);
  rte_prefetch0(src);

  if (likely(ml->off + 6 < ml->m->data_len)) {
    int res = DecodeVarNum(src, 5, n);
    if (likely(res > 0)) {
      ml->off += res;
      ml->rem -= res;
      return NdnError_OK;
    }
    return -res;
  }

  uint8_t firstOctet;
  bool ok = MbufLoc_ReadU8(ml, &firstOctet);
  if (unlikely(!ok)) {
    return NdnError_Incomplete;
  }

  switch (firstOctet) {
    case 253: {
      rte_be16_t v;
      bool ok = MbufLoc_ReadU16(ml, &v);
      if (unlikely(!ok)) {
        return NdnError_Incomplete;
      }
      *n = rte_be_to_cpu_16(v);
      break;
    }
    case 254: {
      rte_be32_t v;
      bool ok = MbufLoc_ReadU32(ml, &v);
      if (unlikely(!ok)) {
        return NdnError_Incomplete;
      }
      *n = rte_be_to_cpu_32(v);
      break;
    }
    case 255:
      return NdnError_LengthOverflow;
    default:
      *n = firstOctet;
      break;
  }
  return NdnError_OK;
}

/** \brief Encode a TLV-TYPE or TLV-LENGTH number.
 *  \param[out] room output buffer, must have \c SizeofVarNum(n) octets
 *  \param n the number
 *  \return room + SizeofVarNum(n)
 */
static __rte_always_inline uint8_t*
EncodeVarNum(uint8_t* room, uint32_t n)
{
  if (likely(n < 253)) {
    room[0] = (uint8_t)n;
    return room + 1;
  }

  if (likely(n <= UINT16_MAX)) {
    room[0] = 253;
    room[1] = (uint8_t)(n >> 8);
    room[2] = (uint8_t)n;
    return room + 3;
  }

  *room++ = 254;
  rte_be32_t v = rte_cpu_to_be_32((uint32_t)n);
  rte_memcpy(room, &v, 4);
  return room + 4;
}

#endif // NDN_DPDK_NDN_TLV_VARNUM_H
