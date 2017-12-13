#ifndef NDN_TRAFFIC_DPDK_NDN_TLV_ELEMENT_H
#define NDN_TRAFFIC_DPDK_NDN_TLV_ELEMENT_H

/// \file

#include "tlv-decoder.h"

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
 *  \param[out] ele the element; will assign all fields except \p last.
 */
static inline NdnError
DecodeTlvHeader(MbufLoc* d, TlvElement* ele, size_t* len)
{
  MbufLoc_Clone(&ele->first, d);

  size_t len1;
  NdnError e = DecodeVarNum(d, &ele->type, &len1);
  *len = len1;
  if (e != NdnError_OK) {
    // no 'unlikely' here: this can commonly occur when d starts at the end
    return e;
  }

  uint64_t tlvLength;
  e = DecodeVarNum(d, &tlvLength, &len1);
  *len += len1;
  if (unlikely(e != NdnError_OK)) {
    return e;
  }
  if (unlikely(tlvLength > UINT32_MAX)) {
    return NdnError_LengthOverflow;
  }
  ele->length = (uint32_t)tlvLength;
  ele->size = *len + ele->length;

  MbufLoc_Clone(&ele->value, d);
  return NdnError_OK;
}

/** \brief Decode a TLV element.
 *  \param[out] ele the element.
 */
static inline NdnError
DecodeTlvElement(MbufLoc* d, TlvElement* ele, size_t* len)
{
  NdnError e = DecodeTlvHeader(d, ele, len);
  if (e != NdnError_OK) {
    return e;
  }

  uint32_t n = MbufLoc_Advance(d, ele->length);
  *len += n;
  if (unlikely(n != ele->length)) {
    return NdnError_Incomplete;
  }

  MbufLoc_Clone(&ele->last, d);
  return NdnError_OK;
}

/** \brief Create a decoder to decode the element's TLV-VALUE.
 *  \param[out] d an iterator bounded inside TLV-VALUE.
 */
static inline void
TlvElement_MakeValueDecoder(const TlvElement* ele, TlvDecoder* d)
{
  MbufLoc_Clone(d, &ele->value);
  d->rem = ele->length;
}

#endif // NDN_TRAFFIC_DPDK_NDN_TLV_ELEMENT_H