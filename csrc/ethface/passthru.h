#ifndef NDNDPDK_ETHFACE_PASSTHRU_H
#define NDNDPDK_ETHFACE_PASSTHRU_H

/** @file */

#include "../iface/rxloop.h"

typedef struct EthGtpip EthGtpip;

enum {
  /// FaceRx/TxThread.nFrames[cntNPkts] counts non-GTP-IP packets.
  EthPassthru_cntNPkts = PktInterest,
  /// FaceRx/TxThread.nFrames[cntNGtpip] counts GTP-IP packets.
  EthPassthru_cntNGtpip = PktData,
};

/**
 * @brief Ethernet pass-through face and its associated TAP port.
 *
 * This struct also serves as the RxGroup of the TAP port.
 */
typedef struct EthPassthru {
  RxGroup base;
  uint16_t tapPort;
  EthGtpip* gtpip;
} EthPassthru;

/**
 * @brief Process a burst of received Ethernet frames on a pass-through face.
 *
 * This is set as @c Face_RxInputFunc of a pass-through face.
 * It transmits every packet out of the associated TAP ethdev.
 */
__attribute__((nonnull)) FaceRxInputResult
EthPassthru_FaceRxInput(Face* face, int rxThread, struct rte_mbuf** pkts, Packet** npkts,
                        uint16_t count);

/**
 * @brief Receive Ethernet frames on a TAP ethdev associated with a pass-through face.
 *
 * This is set as @c RxGroup_RxBurstFunc of a TAP ethdev associated with a pass-through face.
 * It enqueues every packet into the output queue of the pass-through face.
 */
__attribute__((nonnull)) void
EthPassthru_TapPortRxBurst(RxGroup* rxg, RxGroupBurstCtx* ctx);

/**
 * @brief Process outgoing Ethernet frames on a pass-through face.
 *
 * This is set as @c Face_TxLoopFunc of a pass-through face.
 * It transmits every packet out of the (physical) DPDK ethdev.
 */
__attribute__((nonnull)) uint16_t
EthPassthru_TxLoop(Face* face, int txThread);

#endif // NDNDPDK_ETHFACE_PASSTHRU_H
