#ifndef NDN_DPDK_NDN_TLV_DECODE_POS_H
#define NDN_DPDK_NDN_TLV_DECODE_POS_H

/** \file
 *
 *  \par Common parameters of decoding functions:
 *  \param[inout] d the decoder.
 *
 *  \par Common return values of decoding functions:
 *  \retval NdnError_OK successful; decoder is advanced past end of decoded item.
 *  \retval NdnError_Incomplete reaching input boundary before decoding finishes.
 *  \retval NdnError_LengthOverflow TLV-LENGTH is too large.
 *  \retval NdnError_BadType unexpected TLV-TYPE.
 */

#include "common.h"

/** \brief TLV decoder.
 *
 *  The decoder contains an input iterator and boundary.
 */
typedef MbufLoc TlvDecodePos;

static __rte_noinline NdnError
__DecodeVarNum_5or9(TlvDecodePos* d, uint8_t firstOctet, uint32_t* n)
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
      uint64_t number = rte_be_to_cpu_64(v);
      if (unlikely(number > UINT32_MAX)) {
        return NdnError_LengthOverflow;
      }
      *n = (uint32_t)number;
      break;
    }
    default:
      assert(false);
  }
  return NdnError_OK;
}

/** \brief Decode a TLV-TYPE or TLV-LENGTH number.
 *  \param[out] n the number.
 */
static NdnError
DecodeVarNum(TlvDecodePos* d, uint32_t* n)
{
  if (unlikely(MbufLoc_IsEnd(d))) {
    return NdnError_Incomplete;
  }

  uint8_t firstOctet;
  bool ok = MbufLoc_ReadU8(d, &firstOctet);
  if (unlikely(!ok)) {
    return NdnError_Incomplete;
  }

  if (likely(firstOctet < 253)) {
    *n = firstOctet;
    return NdnError_OK;
  }

  if (firstOctet > 253) {
    return __DecodeVarNum_5or9(d, firstOctet, n);
  }

  rte_be16_t v;
  ok = MbufLoc_ReadU16(d, &v);
  if (unlikely(!ok)) {
    return NdnError_Incomplete;
  }
  *n = rte_be_to_cpu_16(v);
  return NdnError_OK;
}

#endif // NDN_DPDK_NDN_TLV_DECODE_POS_H
