#ifndef NDN_DPDK_NDN_TLV_ELEMENT_H
#define NDN_DPDK_NDN_TLV_ELEMENT_H

/// \file

#include "tlv-decode-pos.h"
#include "tlv-type.h"

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

/** \brief Decode a TLV header including TLV-TYPE and TLV-LENGTH but excluding TLV-VALUE.
 *  \param[out] ele the element; will assign all fields except \c last.
 */
static NdnError
DecodeTlvHeader(TlvDecodePos* d, TlvElement* ele)
{
  MbufLoc_Copy(&ele->first, d);

  NdnError e = DecodeVarNum(d, &ele->type);
  RETURN_IF_ERROR; // not unlikely: this occurs when d starts at the end

  uint64_t tlvLength;
  e = DecodeVarNum(d, &tlvLength);
  RETURN_IF_UNLIKELY_ERROR;
  if (unlikely(tlvLength > UINT32_MAX)) {
    return NdnError_LengthOverflow;
  }
  ele->length = (uint32_t)tlvLength;
  ele->size = MbufLoc_FastDiff(&ele->first, d) + ele->length;

  MbufLoc_Copy(&ele->value, d);
  return NdnError_OK;
}

/** \brief Decode a TLV element.
 *  \param[out] ele the element.
 *  \note ele.first.rem, ele.value.rem, and ele.last.rem are unchanged, so that
 *        MbufLoc_FastDiff may be used on them.
 */
static NdnError
DecodeTlvElement(TlvDecodePos* d, TlvElement* ele)
{
  NdnError e = DecodeTlvHeader(d, ele);
  RETURN_IF_ERROR;

  uint32_t n = MbufLoc_Advance(d, ele->length);
  if (unlikely(n != ele->length)) {
    return NdnError_Incomplete;
  }

  MbufLoc_Copy(&ele->last, d);
  return NdnError_OK;
}

/** \brief Decode a TLV element of an expected type.
 *
 *  \retval NdnError_BadType TLV-TYPE does not equal \p expectedType.
 */
static NdnError
DecodeTlvElementExpectType(TlvDecodePos* d, uint64_t expectedType,
                           TlvElement* ele)
{
  NdnError e = DecodeTlvElement(d, ele);
  if (likely(e == NdnError_OK) && unlikely(ele->type != expectedType)) {
    return NdnError_BadType;
  }
  return e;
}

/** \brief Determine if the element's TLV-VALUE is in consecutive memory.
 */
static bool
TlvElement_IsValueLinear(const TlvElement* ele)
{
  return ele->value.off + ele->length <= ele->value.m->data_len;
}

/** \brief Get pointer to element's TLV-VALUE.
 *  \pre TlvElement_IsValueLinear(ele)
 */
static const uint8_t*
TlvElement_GetLinearValue(const TlvElement* ele)
{
  assert(TlvElement_IsValueLinear(ele));
  return rte_pktmbuf_mtod_offset(ele->value.m, const uint8_t*, ele->value.off);
}

/** \brief Ensure TLV-VALUE is in consecutive memory.
 *  \param[inout] ele this TlvElement; TLV-LENGTH must be positive; will be updated.
 *  \param[inout] pkt enclosing packet.
 *  \param mp mempool for copying TLV-VALUE if necessary, requires TLV-LENGTH in dataroom.
 *  \param[out] d a TlvDecodePos pointing to past-end position; NULL if not needed.
 *  \post parent/following TlvElements and TlvDecodePos may be invalidated.
 */
static const uint8_t*
TlvElement_LinearizeValue(TlvElement* ele, struct rte_mbuf* pkt,
                          struct rte_mempool* mp, TlvDecodePos* d)
{
  assert(ele->length > 0);
  const uint8_t* linear = MbufLoc_Linearize(&ele->value, &ele->last, pkt, mp);
  if (d != NULL) {
    // in case MbufLoc_Linearize fails, this is meaningless but harmless
    MbufLoc_Copy(d, &ele->last);
  }
  return linear;
}

/** \brief Create a decoder to decode the element's TLV-VALUE.
 *  \param[out] d an iterator bounded inside TLV-VALUE.
 */
static void
TlvElement_MakeValueDecoder(const TlvElement* ele, TlvDecodePos* d)
{
  MbufLoc_Copy(d, &ele->value);
  d->rem = ele->length;
}

/** \brief Interpret TLV-VALUE as NonNegativeInteger.
 *  \param[out] n the number.
 *  \return whether decoding succeeded
 */
static bool
TlvElement_ReadNonNegativeInteger(const TlvElement* ele, uint64_t* n)
{
  TlvDecodePos vd;
  TlvElement_MakeValueDecoder(ele, &vd);

  switch (ele->length) {
    case 1: {
      uint8_t v;
      bool ok = MbufLoc_ReadU8(&vd, &v);
      if (unlikely(!ok)) {
        return false;
      }
      *n = v;
      return true;
    }
    case 2: {
      rte_be16_t v;
      bool ok = MbufLoc_ReadU16(&vd, &v);
      if (unlikely(!ok)) {
        return false;
      }
      *n = rte_be_to_cpu_16(v);
      return true;
    }
    case 4: {
      rte_be32_t v;
      bool ok = MbufLoc_ReadU32(&vd, &v);
      if (unlikely(!ok)) {
        return false;
      }
      *n = rte_be_to_cpu_32(v);
      return true;
    }
    case 8: {
      rte_be64_t v;
      bool ok = MbufLoc_ReadU64(&vd, &v);
      if (unlikely(!ok)) {
        return false;
      }
      *n = rte_be_to_cpu_64(v);
      return true;
    }
  }

  return false;
}

#endif // NDN_DPDK_NDN_TLV_ELEMENT_H