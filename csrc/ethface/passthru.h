#ifndef NDNDPDK_ETHFACE_PASSTHRU_H
#define NDNDPDK_ETHFACE_PASSTHRU_H

/** @file */

#include "../iface/rxloop.h"

__attribute__((nonnull)) Packet*
EthPassthru_FaceRxInput(Face* face, int rxThread, struct rte_mbuf* pkt);

__attribute__((nonnull)) void
EthPassthru_TapPortRxBurst(RxGroup* rxg, RxGroupBurstCtx* ctx);

__attribute__((nonnull)) uint16_t
EthPassthru_TxLoop(Face* face, int txThread);

#endif // NDNDPDK_ETHFACE_PASSTHRU_H
