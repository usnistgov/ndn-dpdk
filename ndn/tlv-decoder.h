#ifndef NDN_TRAFFIC_DPDK_NDN_TLV_DECODER_H
#define NDN_TRAFFIC_DPDK_NDN_TLV_DECODER_H

/** \file
 *  \brief TLV decoder
 */

#include "common.h"

NdnError __DecodeVarNum_MultiOctet(MbufLoc* ml, uint8_t firstOctet, uint64_t* n,
                                   size_t* len);

/** \brief Decode a TLV-TYPE or TLV-LENGTH number.
 *  \param[inout] ml input iterator.
 *  \param[out] n the number.
 *  \param[out] len decoded length.
 *  \retval NdnError_OK successful.
 *  \retval NdnError_Incomplete reaching input boundary before decoding finishes.
 */
inline NdnError
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