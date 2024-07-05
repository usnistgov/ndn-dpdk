#ifndef NDNDPDK_ETHFACE_FALLBACK_H
#define NDNDPDK_ETHFACE_FALLBACK_H

/** @file */

#include "../iface/rxloop.h"

__attribute__((nonnull)) Packet*
EthFallback_FaceRxInput(Face* face, int rxThread, struct rte_mbuf* pkt);

__attribute__((nonnull)) void
EthFallback_TapPortRxBurst(RxGroup* rxg, RxGroupBurstCtx* ctx);

__attribute__((nonnull)) uint16_t
EthFallback_TxLoop(Face* face, int txThread);

#endif // NDNDPDK_ETHFACE_FALLBACK_H
