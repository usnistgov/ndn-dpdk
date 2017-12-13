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

/** \brief TLV element
 */
typedef struct TlvElement
{
  uint64_t type;   ///< TLV-TYPE number
  uint32_t length; ///< TLV-LENGTH
  uint32_t size;   ///< total length
  MbufLoc first;   ///< start position
  MbufLoc value;   ///< TLV-VALUE position
  MbufLoc last;    ///< past end position
} TlvElement;
static_assert(sizeof(TlvElement) <= RTE_CACHE_LINE_SIZE, ""); // keep it small

/** \brief Decode a TLV header including TLV-TYPE and TLV-LENGTH but excluding TLV-VALUE.
 *  \param[inout] ml input iterator; will be advanced past the end of TLV-LENGTH.
 *  \param[out] ele the element; will assign all fields except \p last.
 *  \param[out] len decoded length.
 *  \retval NdnError_OK successful.
 *  \retval NdnError_Incomplete reaching input boundary before decoding finishes.
 *  \retval NdnError_TlvLengthOverflow TLV-LENGTH is too large.
 */
static inline NdnError
DecodeTlvHeader(MbufLoc* ml, TlvElement* ele, size_t* len)
{
  MbufLoc_Clone(&ele->first, ml);

  size_t len1;
  NdnError e = DecodeVarNum(ml, &ele->type, &len1);
  *len = len1;
  if (e != NdnError_OK) {
    // no 'unlikely' here: this can commonly occur when ml starts at the end
    return e;
  }

  uint64_t tlvLength;
  e = DecodeVarNum(ml, &tlvLength, &len1);
  *len += len1;
  if (unlikely(e != NdnError_OK)) {
    return e;
  }
  if (unlikely(tlvLength > UINT32_MAX)) {
    return NdnError_TlvLengthOverflow;
  }
  ele->length = (uint32_t)tlvLength;
  ele->size = *len + ele->length;

  MbufLoc_Clone(&ele->value, ml);
  return NdnError_OK;
}

/** \brief Decode a TLV element.
 *  \param[inout] ml input iterator; will be advanced past the end of TLV-LENGTH.
 *  \param[out] ele the element.
 *  \param[out] len decoded length.
 *  \retval NdnError_OK successful.
 *  \retval NdnError_Incomplete reaching input boundary before decoding finishes.
 *  \retval NdnError_TlvLengthOverflow TLV-LENGTH is too large.
 */
static inline NdnError
DecodeTlvElement(MbufLoc* ml, TlvElement* ele, size_t* len)
{
  NdnError e = DecodeTlvHeader(ml, ele, len);
  if (e != NdnError_OK) {
    return e;
  }

  uint32_t n = MbufLoc_Advance(ml, ele->length);
  *len += n;
  if (unlikely(n != ele->length)) {
    return NdnError_Incomplete;
  }

  MbufLoc_Clone(&ele->last, ml);
  return NdnError_OK;
}

#endif // NDN_TRAFFIC_DPDK_NDN_TLV