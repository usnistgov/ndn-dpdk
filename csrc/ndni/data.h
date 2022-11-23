#ifndef NDNDPDK_NDNI_DATA_H
#define NDNDPDK_NDNI_DATA_H

/** @file */

#include "name.h"

/** @brief Parsed Data packet. */
typedef struct PData
{
  PName name;
  uint32_t freshness; ///< FreshnessPeriod in millis
  bool hasDigest;
  bool isFinalBlock;
  uint32_t contentOffset; ///< Content TLV-VALUE offset
  uint32_t contentL;      ///< Content TLV-LENGTH
  RTE_MARKER64 a_;
  uint8_t digest[ImplicitDigestLength];
  RTE_MARKER64 b_;
  uint8_t helperScratch[192]; ///< scratch area for helper threads
} PData;

/**
 * @brief Parse Data.
 * @param pkt a uniquely owned, possibly segmented, direct mbuf that contains Data TLV.
 * @param parseFor if set to @c ParseForFw , skip FinalBlock and set @c data->isFinalBlock to false.
 * @return whether success.
 */
__attribute__((nonnull)) bool
PData_Parse(PData* data, struct rte_mbuf* pkt, ParseFor parseFor);

/** @brief Determine whether Data can satisfy Interest. */
__attribute__((nonnull)) DataSatisfyResult
PData_CanSatisfy(PData* data, PInterest* interest);

/**
 * @brief Prepare a crypto_op for Data digest computation.
 * @param npkt Data packet.
 * @return rte_crypto_op placed in PData.helperScratch.
 */
__attribute__((nonnull, returns_nonnull)) struct rte_crypto_op*
DataDigest_Prepare(CryptoQueuePair* cqp, Packet* npkt);

/**
 * @brief Enqueue crypto_ops for Data digest computation.
 * @return number of rejected packets; they should be freed by caller.
 */
__attribute__((nonnull)) uint16_t
DataDigest_Enqueue(CryptoQueuePair* cqp, struct rte_crypto_op** ops, uint16_t count);

/**
 * @brief Finish Data digest computation.
 * @param op a dequeued crypto_op; will be freed.
 * @param[out] npkt the Data packet; DataDigest releases ownership.
 * @return whether success.
 */
__attribute__((nonnull)) bool
DataDigest_Finish(struct rte_crypto_op* op, Packet** npkt);

/**
 * @brief Prepare Data MetaInfo.
 * @param room output buffer; must have enough capacity.
 * @param ct ContentType numeric value.
 * @param freshness FreshnessPeriod numeric value.
 * @param finalBlock FinalBlockId TLV-VALUE.
 * @post @c room contains MetaInfo TLV.
 *
 * Required @p room capacity is the sum of:
 * @li MetaInfo TLV-TYPE and TLV-LENGTH, 2 octets
 * @li ContentType TLV, 3 octets.
 * @li FreshnessPeriod TLV, 6 octets.
 * @li FinalBlockId TLV, 2 octets + @c finalBlock.length .
 */
__attribute__((nonnull)) void
DataEnc_PrepareMetaInfo(uint8_t* room, ContentType ct, uint32_t freshness, LName finalBlock);

/**
 * @brief Returned size of MetaInfo TLV.
 * @param meta prepared MetaInfo buffer.
 */
__attribute__((nonnull)) inline uint16_t
DataEnc_SizeofMetaInfo(const uint8_t* meta)
{
  return 2 + meta[1];
}

/**
 * @brief Encode Data with payload.
 * @param prefix Data name prefix.
 * @param suffix Data name suffix.
 * @param meta prepared MetaInfo buffer.
 * @param m a uniquely owned, unsegmented, direct mbuf of Content payload.
 * @return encoded packet, same as @p m .
 * @retval NULL insufficient headroom or tailroom.
 */
__attribute__((nonnull)) Packet*
DataEnc_EncodePayload(LName prefix, LName suffix, const uint8_t* meta, struct rte_mbuf* m);

/** @brief Data encoder optimized for traffic generator. */
typedef struct DataGen
{
  /**
   * @brief Template mbuf.
   *
   * This should contain name suffix TLV-VALUE and fields after Name.
   * Name TL and Data TL are not included.
   */
  struct rte_mbuf* tpl;

  /** @brief Size of name suffix TLV-VALUE at the beginning of @c tpl . */
  uint16_t suffixL;
} DataGen;

/**
 * @brief Encode Data with DataGen template.
 * @return encoded packet.
 * @retval NULL allocation failure.
 *
 * If @c align.linearize is false, encoded packet has a header mbuf that contains @p prefix and
 * and an indirect mbuf that clones the template. @c mp->header dataroom must be at least
 * @c RTE_PKTMBUF_DATAROOM+LpHeaderHeadroom+DataGenDataroom .
 *
 * If @c align.linearize is true, encoded packet has one or more copied mbufs. @c mp->packet
 * dataroom must be at least @c RTE_PKTMBUF_DATAROOM+LpHeaderHeadroom+align.fragmentPayloadSize .
 */
__attribute__((nonnull)) Packet*
DataGen_Encode(DataGen* gen, LName prefix, PacketMempools* mp, PacketTxAlign align);

#endif // NDNDPDK_NDNI_DATA_H
