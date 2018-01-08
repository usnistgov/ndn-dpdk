#ifndef NDN_DPDK_NDN_TLV_DECODER_H
#define NDN_DPDK_NDN_TLV_DECODER_H

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
typedef MbufLoc TlvDecoder;

NdnError __DecodeVarNum_32or64(TlvDecoder* d, uint8_t firstOctet, uint64_t* n);

/** \brief Decode a TLV-TYPE or TLV-LENGTH number.
 *  \param[out] n the number.
 */
static inline NdnError
DecodeVarNum(TlvDecoder* d, uint64_t* n)
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
    return __DecodeVarNum_32or64(d, firstOctet, n);
  }

  rte_be16_t v;
  ok = MbufLoc_ReadU16(d, &v);
  if (unlikely(!ok)) {
    return NdnError_Incomplete;
  }
  *n = rte_be_to_cpu_16(v);
  return NdnError_OK;
}

#endif // NDN_DPDK_NDN_TLV_DECODER_H
