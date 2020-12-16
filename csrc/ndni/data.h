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
  uint8_t digest[32];
} PData;

/**
 * @brief Parse Data.
 * @param pkt a uniquely owned, possibly segmented, direct mbuf that contains Data TLV.
 * @return whether success.
 */
__attribute__((nonnull)) bool
PData_Parse(PData* data, struct rte_mbuf* pkt);

/** @brief Determine whether Data can satisfy Interest. */
__attribute__((nonnull)) DataSatisfyResult
PData_CanSatisfy(PData* data, PInterest* interest);

/**
 * @brief Prepare a crypto_op for Data digest computation.
 * @param npkt Data packet.
 * @param[out] op an allocated crypto_op; will be populated but not enqueued.
 */
__attribute__((nonnull)) void
DataDigest_Prepare(Packet* npkt, struct rte_crypto_op* op);

/**
 * @brief Finish Data digest computation.
 * @param op a dequeued crypto_op; will be freed.
 * @return the Data packet, or NULL if crypto_op was unsuccessful.
 */
__attribute__((nonnull)) Packet*
DataDigest_Finish(struct rte_crypto_op* op);

typedef struct DataGen DataGen;

typedef Packet* (*DataGen_EncodeFunc)(DataGen* gen, LName prefix, PacketMempools* mp);

/**
 * @brief Data encoder optimized for traffic generator.
 *
 * DataGen* is struct rte_mbuf*.
 * Its packet buffer contains name suffix TLV-VALUE and fields after Name.
 * Name TL and Data TL are not included.
 */
typedef struct DataGen
{
  struct rte_mbuf* tpl;
  uint16_t suffixL;
  DataGen_EncodeFunc encode;
} DataGen;

__attribute__((nonnull)) void
DataGen_Init(DataGen* gen, PacketTxAlign align);

/**
 * @brief Encode Data with DataGen template.
 * @param mp @c mp->header should have RTE_PKTMBUF_DATAROOM + LpHeaderHeadroom + DataGenDataroom.
 * @return encoded packet.
 * @retval NULL allocation failure.
 */
__attribute__((nonnull)) static inline Packet*
DataGen_Encode(DataGen* gen, LName prefix, PacketMempools* mp)
{
  return (gen->encode)(gen, prefix, mp);
}

#endif // NDNDPDK_NDNI_DATA_H
