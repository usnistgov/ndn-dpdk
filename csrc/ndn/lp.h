#ifndef NDN_DPDK_NDN_LP_H
#define NDN_DPDK_NDN_LP_H

/** @file */

#include "tlv-element.h"

/** @brief NDNLPv2 layer 2 fields. */
typedef struct LpL2
{
  uint64_t seqNum;
  uint16_t fragIndex;
  uint16_t fragCount;
} LpL2;

/** @brief NDNLPv2 layer 3 fields. */
typedef struct LpL3
{
  uint64_t pitToken;
  uint8_t nackReason;
  uint8_t congMark;
} LpL3;

/** @brief Parsed NDNLPv2 header. */
typedef struct LpHeader
{
  LpL3 l3;
  LpL2 l2;
} LpHeader;

/**
 * @brief Parse a packet as NDNLPv2.
 * @param[out] lph the parsed LpHeader.
 * @param pkt the packet.
 * @param[out] payloadOff payload offset.
 * @param[out] tlvSize size of top-level TLV.
 *
 * This function recognizes these NDNLPv2 features:
 * @li indexed fragmentation-reassembly
 * @li network nack
 * @li congestion mark
 *
 * This function does not check whether header fields are applicable to network layer packet type,
 * because network layer type is unknown before reassembly. For example, it would accept Nack
 * header on Data packet.
 *
 * @retval NdnErrBadType packet is not LpPacket or bare Interest/Data.
 * @retval NdnErrLengthOverflow FragIndex, FragCount, NackReason, or CongestionMark
 *         number is too large to be stored in the header field.
 * @retval NdnErrFragIndexExceedFragCount FragIndex is not less than FragCount.
 * @retval NdnErrLpHasTrailer found trailer fields after LpFragment.
 */
NdnError
LpHeader_FromPacket(LpHeader* lph, struct rte_mbuf* pkt, uint32_t* payloadOff, uint32_t* tlvSize);

/**
 * @brief Encode LP header in headroom.
 * @param m output mbuf, must be first segment, and must have
 *          @c LpHeaderEstimatedHeadroom in headroom.
 * @param payloadL TLV-LENGTH of LpPayload, or 0 to indicate no payload
 */
void
PrependLpHeader(struct rte_mbuf* m, const LpHeader* lph, uint32_t payloadL);

#endif // NDN_DPDK_NDN_LP_H
