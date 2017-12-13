#ifndef NDN_TRAFFIC_DPDK_NDN_TLV_DECODER_H
#define NDN_TRAFFIC_DPDK_NDN_TLV_DECODER_H

/** \file
 *
 *  \par Common parameters of decoding functions:
 *  \param[inout] d the decoder.
 *  \param[out] len total length of decoded item.
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

NdnError __DecodeVarNum_MultiOctet(TlvDecoder* d, uint8_t firstOctet,
                                   uint64_t* n, size_t* len);

/** \brief Decode a TLV-TYPE or TLV-LENGTH number.
 *  \param[out] n the number.
 */
static inline NdnError
DecodeVarNum(TlvDecoder* d, uint64_t* n, size_t* len)
{
  if (unlikely(MbufLoc_IsEnd(d))) {
    return NdnError_Incomplete;
  }

  uint8_t firstOctet;
  bool ok = MbufLoc_ReadU8(d, &firstOctet);
  if (unlikely(!ok)) {
    return NdnError_Incomplete;
  }

  if (unlikely(firstOctet >= 253)) {
    return __DecodeVarNum_MultiOctet(d, firstOctet, n, len);
  }

  *len = 1;
  *n = firstOctet;
  return NdnError_OK;
}

#endif // NDN_TRAFFIC_DPDK_NDN_TLV