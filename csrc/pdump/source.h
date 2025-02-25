#ifndef NDNDPDK_PDUMP_SOURCE_H
#define NDNDPDK_PDUMP_SOURCE_H

/** @file */

#include "../core/urcu.h"
#include "../iface/faceid.h"
#include "../vendor/pcg_basic.h"
#include "enum.h"
#include <urcu-pointer.h>

typedef struct PdumpSource PdumpSource;

typedef bool (*PdumpSource_Filter)(PdumpSource* s, struct rte_mbuf* pkt);

/** @brief Packet dump source. */
struct PdumpSource {
  struct rte_mempool* directMp;
  struct rte_ring* queue;
  PdumpSource_Filter filter;
  uint32_t mbufType;
  uint16_t mbufPort;
  bool mbufCopy;
};

/** @brief Submit packets for potential dumping. */
__attribute__((nonnull)) void
PdumpSource_Process(PdumpSource* s, struct rte_mbuf** pkts, uint16_t count);

/** @brief RCU-protected pointer to PdumpSource. */
typedef struct PdumpSourceRef {
  PdumpSource* s;
} PdumpSourceRef;

/**
 * @brief Assign or clear PdumpSource in PdumpSourceRef.
 * @return old pointer value.
 */
__attribute__((nonnull(1))) PdumpSource*
PdumpSourceRef_Set(PdumpSourceRef* ref, PdumpSource* s);

/**
 * @brief Retrieve dumper if enabled.
 * @return the dumper, NULL if dumper is disabled.
 * @pre Calling thread holds rcu_read_lock.
 */
__attribute__((nonnull)) static __rte_always_inline PdumpSource*
PdumpSourceRef_Get(PdumpSourceRef* ref) {
  return rcu_dereference(ref->s);
}

/**
 * @brief Submit packets for potential dumping if dumper is enabled.
 * @pre Calling thread holds rcu_read_lock.
 */
__attribute__((nonnull)) static __rte_always_inline bool
PdumpSourceRef_Process(PdumpSourceRef* ref, struct rte_mbuf** pkts, uint16_t count) {
  PdumpSource* s = PdumpSourceRef_Get(ref);
  if (s == NULL) {
    return false;
  }
  PdumpSource_Process(s, pkts, count);
  return true;
}

/** @brief Packet dump from a face on RX or TX direction. */
typedef struct PdumpFaceSource {
  PdumpSource base;
  pcg32_random_t rng;
  uint32_t sample[PdumpMaxNames];
  uint16_t nameL[PdumpMaxNames];
  uint8_t nameV[PdumpMaxNames * NameMaxLength];
} PdumpFaceSource;

/**
 * @brief Perform name filtering to determine whether to capture a packet.
 * @param pkt packet positioned at NDNLPv2 header.
 * @return whether to capture the packet.
 */
__attribute__((nonnull)) bool
PdumpFaceSource_Filter(PdumpSource* s, struct rte_mbuf* pkt);

extern PdumpSourceRef gPdumpEthPortSources[RTE_MAX_ETHPORTS];

/** @brief Packet dump for unmatched frames on an Ethernet port, contextual information. */
typedef struct PdumpEthPortUnmatchedCtx {
  PdumpSource* source;
  uint16_t count;
  struct rte_mbuf* pkts[MaxBurstSize];
} PdumpEthPortUnmatchedCtx;

/** @brief Initialize PdumpEthPortUnmatchedCtx to be disabled. */
static __rte_always_inline void
PdumpEthPortUnmatchedCtx_Disable(PdumpEthPortUnmatchedCtx* ctx) {
  ctx->source = NULL;
  ctx->count = 0;
  POISON(ctx->pkts);
}

/**
 * @brief Initialize PdumpEthPortUnmatchedCtx for an ethdev.
 * @pre Calling thread holds rcu_read_lock.
 */
static __rte_always_inline void
PdumpEthPortUnmatchedCtx_Init(PdumpEthPortUnmatchedCtx* ctx, uint16_t port) {
  PdumpSourceRef* ref = &gPdumpEthPortSources[port];
  ctx->source = PdumpSourceRef_Get(ref);
  ctx->count = 0;
  POISON(ctx->pkts);
}

/**
 * @brief Append an Ethernet frame to be captured.
 * @param pkt Ethernet frame.
 * @retval true packet is accepted and owned by pdump.
 * @retval false packet is rejected and should be freed by caller.
 */
static __rte_always_inline bool
PdumpEthPortUnmatchedCtx_Append(PdumpEthPortUnmatchedCtx* ctx, struct rte_mbuf* pkt) {
  if (ctx->source == NULL) {
    return false;
  }
  ctx->pkts[ctx->count++] = pkt;
  return true;
}

/** @brief Submit accumulated packets to the pdump writer. */
static __rte_always_inline void
PdumpEthPortUnmatchedCtx_Process(PdumpEthPortUnmatchedCtx* ctx) {
  if (ctx->source == NULL || ctx->count == 0) {
    return;
  }
  PdumpSource_Process(ctx->source, ctx->pkts, ctx->count);
  NULLize(ctx->source);
  POISON(ctx->pkts);
}

#endif // NDNDPDK_PDUMP_SOURCE_H
