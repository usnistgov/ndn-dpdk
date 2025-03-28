#ifndef NDNDPDK_ETHFACE_RXMATCH_H
#define NDNDPDK_ETHFACE_RXMATCH_H

/** @file */

#include "locator.h"

typedef enum EthRxMatchAct {
  EthRxMatchActAlways = 1,
  EthRxMatchActEtherUnicast,
  EthRxMatchActEtherMulticast,
  EthRxMatchActUdp,
  EthRxMatchActVxlan,
  EthRxMatchActGtp,
} __rte_packed EthRxMatchAct;

/** @brief Bit flags in @c EthRxMatch_Match return value. */
typedef enum EthRxMatchResult {
  EthRxMatchResultHit = RTE_BIT32(1), ///< fully matched
  EthRxMatchResultGtp = RTE_BIT32(2), ///< GTP-U tunnel matched
} __rte_packed EthRxMatchResult;

typedef struct EthRxMatch EthRxMatch;
typedef EthRxMatchResult (*EthRxMatch_MatchFunc)(const EthRxMatch* match, const struct rte_mbuf* m);
extern const EthRxMatch_MatchFunc EthRxMatch_MatchJmp[];

/** @brief EthFace RX matcher. */
struct EthRxMatch {
  EthRxMatchAct act;  ///< EthRxMatch_MatchJmp index
  uint8_t len;        ///< total header length
  uint8_t l2len;      ///< outer Ethernet+VLAN length
  uint8_t l3matchOff; ///< offset of outer IPv4/IPv6 src
  uint8_t l3matchLen; ///< length of IP src+dst, plus UDP src+dst if checked
  uint8_t udpOff;     ///< offset of outer UDP header
  uint8_t buf[EthLocator_MaxHdrLen];
};

/** @brief Prepare RX matcher from locator. */
__attribute__((nonnull)) void
EthRxMatch_Prepare(EthRxMatch* match, const EthLocator* loc);

/**
 * @brief Determine whether a received frame matches the locator.
 * @param match EthRxMatch prepared by @c EthRxMatch_Prepare .
 */
__attribute__((nonnull)) static inline EthRxMatchResult
EthRxMatch_Match(const EthRxMatch* match, const struct rte_mbuf* m) {
  if (m->data_len < match->len) {
    return 0;
  }
  return EthRxMatch_MatchJmp[match->act](match, m);
}

/**
 * @brief Check GTP-U inner headers only.
 * @param match EthRxMatch prepared from GTP-U locator.
 * @param m mbuf with sufficient data_len.
 */
__attribute__((nonnull)) EthRxMatchResult
EthRxMatch_MatchGtpInner(const EthRxMatch* match, const struct rte_mbuf* m);

#endif // NDNDPDK_ETHFACE_RXMATCH_H
