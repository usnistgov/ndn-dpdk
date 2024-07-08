#ifndef NDNDPDK_ETHFACE_PASSTHRU_H
#define NDNDPDK_ETHFACE_PASSTHRU_H

/** @file */

#include "../iface/rxloop.h"

/**
 * @brief Process incoming Ethernet frame on a pass-through face.
 *
 * This is set as @c Face_RxInputFunc of a pass-through face.
 * It transmits every packet out of the associated TAP ethdev.
 */
__attribute__((nonnull)) Packet*
EthPassthru_FaceRxInput(Face* face, int rxThread, struct rte_mbuf* pkt);

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
