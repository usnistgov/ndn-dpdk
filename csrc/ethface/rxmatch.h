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
} EthRxMatchAct;

typedef struct EthRxMatch EthRxMatch;
typedef bool (*EthRxMatch_MatchFunc)(const EthRxMatch* match, const struct rte_mbuf* m);
extern const EthRxMatch_MatchFunc EthRxMatch_MatchJmp[];

/** @brief EthFace RX matcher. */
struct EthRxMatch {
  EthRxMatchAct act;
  uint8_t len;
  uint8_t l2len;
  uint8_t l3matchOff;
  uint8_t l3matchLen;
  uint8_t udpOff;
  uint8_t buf[EthFace_HdrMax];
};

/** @brief Prepare RX matcher from locator. */
__attribute__((nonnull)) void
EthRxMatch_Prepare(EthRxMatch* match, const EthLocator* loc);

/**
 * @brief Determine whether a received frame matches the locator.
 * @param match EthRxMatch prepared by @c EthRxMatch_Prepare .
 */
__attribute__((nonnull)) static inline bool
EthRxMatch_Match(const EthRxMatch* match, const struct rte_mbuf* m) {
  return m->data_len >= match->len && EthRxMatch_MatchJmp[match->act](match, m);
}

#endif // NDNDPDK_ETHFACE_RXMATCH_H
