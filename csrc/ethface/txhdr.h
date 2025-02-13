#ifndef NDNDPDK_ETHFACE_TXHDR_H
#define NDNDPDK_ETHFACE_TXHDR_H

/** @file */

#include "locator.h"

typedef struct EthTxHdr EthTxHdr;

/** @brief EthFace TX header template. */
struct EthTxHdr {
  void (*f)(const EthTxHdr* hdr, struct rte_mbuf* m, bool newBurst);
  uint8_t len;
  uint8_t l2len;
  char tunnel;
  uint8_t buf[EthFace_HdrMax];
};

/** @brief Prepare TX header from locator. */
__attribute__((nonnull)) void
EthTxHdr_Prepare(EthTxHdr* hdr, const EthLocator* loc, bool hasChecksumOffloads);

/**
 * @brief Prepend TX header.
 * @param hdr prepared by @c EthTxHdr_Prepare .
 * @param newBurst whether @p m is the first frame in a new burst.
 */
__attribute__((nonnull)) static inline void
EthTxHdr_Prepend(const EthTxHdr* hdr, struct rte_mbuf* m, bool newBurst) {
  hdr->f(hdr, m, newBurst);
}

#endif // NDNDPDK_ETHFACE_TXHDR_H
