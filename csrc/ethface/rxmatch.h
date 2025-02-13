#ifndef NDNDPDK_ETHFACE_RXMATCH_H
#define NDNDPDK_ETHFACE_RXMATCH_H

/** @file */

#include "locator.h"

typedef struct EthRxMatch EthRxMatch;

/** @brief EthFace RX matcher. */
struct EthRxMatch {
  bool (*f)(const EthRxMatch* match, const struct rte_mbuf* m);
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
  return m->data_len >= match->len && match->f(match, m);
}

#endif // NDNDPDK_ETHFACE_RXMATCH_H
