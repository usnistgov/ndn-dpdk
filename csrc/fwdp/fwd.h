#ifndef NDNDPDK_FWDP_FWD_H
#define NDNDPDK_FWDP_FWD_H

/** @file */

#include "../core/running-stat.h"
#include "../dpdk/thread.h"
#include "../fib/fib.h"
#include "../fib/nexthop-filter.h"
#include "../iface/face.h"
#include "../iface/pktqueue.h"
#include "../pcct/cs.h"
#include "../pcct/pit.h"
#include "../strategyapi/api.h"

typedef struct FwFwdCtx FwFwdCtx;

/** @brief Forwarding thread. */
typedef struct FwFwd {
  SgGlobal sgGlobal;
  ThreadCtrl ctrl;
  PktQueue queueI;
  PktQueue queueD;
  PktQueue queueN;

  Fib* fib;
  Pit* pit;
  Cs* cs;

  pcg32_random_t sgRng;
  PitSuppressConfig suppressCfg;

  uint8_t id;          ///< fwd process id
  uint8_t fibDynIndex; ///< FibEntry.dyn index

  uint64_t nNoFibMatch;   ///< Interests dropped due to no FIB match
  uint64_t nDupNonce;     ///< Interests dropped due to duplicate nonce
  uint64_t nSgNoFwd;      ///< Interests not forwarded by strategy
  uint64_t nNackMismatch; ///< Nack dropped due to outdated nonce

  PacketMempools mp; ///< mempools for packet modification

  struct rte_ring* cryptoHelper; ///< queue to crypto helper

  /** @brief Statistics of latency from packet arrival to start processing. */
  RunningStat latencyStat;
} FwFwd;

__attribute__((nonnull)) int
FwFwd_Run(FwFwd* fwd);

__attribute__((nonnull)) void
FwFwd_RxInterest(FwFwd* fwd, FwFwdCtx* ctx);

__attribute__((nonnull)) void
FwFwd_RxData(FwFwd* fwd, FwFwdCtx* ctx);

__attribute__((nonnull)) void
FwFwd_RxNack(FwFwd* fwd, FwFwdCtx* ctx);

/**
 * @brief Per-packet context in forwarding.
 *
 * Field availability:
 * T: set by SgTriggerTimer and available during SGEVT_TIMER
 * F: set by FwFwd_Run
 * I: available during SGEVT_INTEREST
 * D: available during SGEVT_DATA
 * N: available during SGEVT_NACK
 */
struct FwFwdCtx {
  FwFwd* fwd;             // T,F,I,D,N
  TscTime rxTime;         // T(=now),F,I,D,N
  SgEvent eventKind;      // T,F,I,D,N
  FibNexthopFilter nhFlt; // T,I,D,N
  union {
    Packet* npkt;
    struct rte_mbuf* pkt;
  };                        // F,D,N
  FibEntry* fibEntry;       // T,I,D,N
  FibEntryDyn* fibEntryDyn; // T,I,D,N
  PitEntry* pitEntry;       // T,I,D,N

  // end of SgCtx fields
  RTE_MARKER endofSgCtx;

  PitUp* pitUp;       // N
  LpPitToken rxToken; // F,I,D,N
  uint32_t dnNonce;   // I
  int nForwarded;     // T,I,N
  FaceID rxFace;      // F,I,D
};

/** @brief Free the current @c npkt . */
__attribute__((nonnull)) static inline void
FwFwdCtx_FreePkt(FwFwdCtx* ctx) {
  Packet_Free(ctx->npkt);
  NULLize(ctx->npkt);
}

/** @brief Assign @c fibEntry and @c fibEntryDyn . */
__attribute__((nonnull(1))) static inline void
FwFwdCtx_SetFibEntry(FwFwdCtx* ctx, FibEntry* fibEntry) {
  ctx->fibEntry = fibEntry;
  if (likely(fibEntry != NULL)) {
    ctx->fibEntryDyn = FibEntry_PtrDyn(fibEntry, ctx->fwd->fibDynIndex);
  } else {
    ctx->fibEntryDyn = NULL;
  }
}

#endif // NDNDPDK_FWDP_FWD_H
