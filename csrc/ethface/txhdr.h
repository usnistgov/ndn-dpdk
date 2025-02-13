#ifndef NDNDPDK_ETHFACE_TXHDR_H
#define NDNDPDK_ETHFACE_TXHDR_H

/** @file */

#include "locator.h"

typedef enum EthTxHdrAct {
  EthTxHdrActNoHdr = 1,
  EthTxHdrActEther,

  EthTxHdrActUdp4Checksum = 0b1010,
  EthTxHdrActUdp4Offload = 0b1011,
  EthTxHdrActUdp6Checksum = 0b1000,
  EthTxHdrActUdp6Offload = 0b1001,
} EthTxHdrAct;

/** @brief Bit flags for @c EthTxHdr_Prepend . */
typedef enum EthTxHdrFlags {
  /** @brief Whether mbuf is the first frame in a new burst. */
  EthTxHdrFlagsNewBurst = RTE_BIT32(0),
  /** @brief Whether mbuf contains Ethernet+IPv4 instead of NDN. */
  EthTxHdrFlagsGtpip = RTE_BIT32(1),
} EthTxHdrFlags;

typedef struct EthTxHdr EthTxHdr;
typedef void (*EthTxHdr_PrependFunc)(const EthTxHdr* hdr, struct rte_mbuf* m, EthTxHdrFlags flags);
extern const EthTxHdr_PrependFunc EthTxHdr_PrependJmp[];

/** @brief EthFace TX header template. */
struct EthTxHdr {
  EthTxHdrAct act;
  uint8_t len;
  uint8_t l2len;
  char tunnel;
  uint8_t buf[EthLocator_MaxHdrLen];
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
EthTxHdr_Prepend(const EthTxHdr* hdr, struct rte_mbuf* m, EthTxHdrFlags flags) {
  EthTxHdr_PrependJmp[hdr->act](hdr, m, flags);
}

#endif // NDNDPDK_ETHFACE_TXHDR_H
