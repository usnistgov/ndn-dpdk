#ifndef NDN_DPDK_NDN_LP_PKT_H
#define NDN_DPDK_NDN_LP_PKT_H

/// \file

#include "tlv-element.h"

/** \brief TLV LpPacket
 */
typedef struct LpPkt
{
  uint64_t pitToken;
  uint64_t seqNo;
  uint16_t fragIndex;
  uint16_t fragCount;
  uint8_t nackReason;
  uint8_t congMark;

  uint16_t payloadOff; ///< offset of payload
  MbufLoc payload;     ///< start position and boundary of payload
} LpPkt;

/** \brief Decode an LpPacket.
 *  \param[out] lpp the LpPacket.
 *
 *  This function recognizes these NDNLPv2 features:
 *  \li indexed fragmentation-reassembly
 *  \li network nack
 *  \li congestion mark
 *
 *  This function does not check whether header fields are applicable to network layer packet type,
 *  because network layer type is unknown before reassembly. For example, it will accept Nack
 *  header on Data packet.
 *
 *  \retval NdnError_LengthOverflow FragIndex, FragCount, NackReason, or CongestionMark number is
 *          too large to be stored in the header.
 *  \retval NdnError_FragIndexExceedFragCount FragIndex is not less than FragCount.
 */
NdnError DecodeLpPkt(TlvDecoder* d, LpPkt* lpp);

/** \brief Test whether \p lpp contains payload.
 */
static bool
LpPkt_HasPayload(const LpPkt* lpp)
{
  return !MbufLoc_IsEnd(&lpp->payload);
}

/** \brief Test whether the payload of \p lpp is fragmented.
 */
static bool
LpPkt_IsFragmented(const LpPkt* lpp)
{
  return lpp->fragCount > 1;
}

static uint16_t
EncodeLpHeaders_GetHeadroom()
{
  return 1 + 5;
}

static uint16_t
EncodeLpHeaders_GetTailroom()
{
  return 1 + 1 + 8 +         // SeqNo
         1 + 1 + 2 +         // FragIndex
         1 + 1 + 2 +         // FragCount
         1 + 1 + 8 +         // PitToken
         3 + 1 + 3 + 1 + 1 + // Nack
         3 + 1 + 1 +         // CongestionMark
         1 + 5;              // Payload
}

/** \brief Encode LP headers.
 *  \param m output mbuf, must be empty and is the only segment, must have
 *           \p EncodeLpHeaders_GetHeadroom() in headroom and
 *           \p EncodeLpHeaders_GetTailroom() in tailroom
 *  \param lpp LP header, must have payload
 *
 *  This function writes LpPacket's TLV-TYPE and TLV-LENGTH to headroom, and writes
 *  other header fields as well as Payload's TLV-TYPE and TLV-LENGTH to tailroom.
 *  If \p LpPkt_IsFragmented(lpp) , this function skips Nack and CongestionMark fields.
 */
void EncodeLpHeaders(struct rte_mbuf* m, const LpPkt* lpp);

#endif // NDN_DPDK_NDN_LP_PKT_H