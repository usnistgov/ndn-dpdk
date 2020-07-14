#ifndef NDNDPDK_FWDP_FWD_H
#define NDNDPDK_FWDP_FWD_H

/** @file */

#include "../core/running-stat.h"
#include "../dpdk/thread.h"
#include "../fib/fib.h"
#include "../iface/face.h"
#include "../iface/pktqueue.h"
#include "../pcct/cs.h"
#include "../pcct/pit.h"
#include "../strategyapi/api.h"

/** @brief Forwarding thread. */
typedef struct FwFwd
{
  SgGlobal sgGlobal;
  PktQueue queueI;
  PktQueue queueD;
  PktQueue queueN;

  Fib* fib;
  union
  {
    Pcct* pcct;
    Pit* pit;
    Cs* cs;
  };

  PitSuppressConfig suppressCfg;

  uint8_t id; ///< fwd process id
  ThreadStopFlag stop;

  uint64_t nNoFibMatch;   ///< Interests dropped due to no FIB match
  uint64_t nDupNonce;     ///< Interests dropped due duplicate nonce
  uint64_t nSgNoFwd;      ///< Interests not forwarded by strategy
  uint64_t nNackMismatch; ///< Nack dropped due to outdated nonce

  struct rte_mempool* headerMp;   ///< mempool for Interest/Data header/guider
  struct rte_mempool* indirectMp; ///< mempool for indirect mbufs

  struct rte_ring* crypto; ///< queue to crypto helper

  /** @brief Statistics of latency from packet arrival to start processing. */
  RunningStat latencyStat;
} FwFwd;

static inline Pcct**
FwFwd_GetPcctPtr_(FwFwd* fwd)
{
  return &fwd->pcct;
}

int
FwFwd_Run(FwFwd* fwd);

/**
 * @brief Per-packet context in forwarding.
 *
 * Field availablility:
 * T: set by SgTriggerTimer and available during SGEVT_TIMER
 * F: set by FwFwd_Run
 * I: available during SGEVT_INTEREST
 * D: available during SGEVT_DATA
 * N: available during SGEVT_NACK
 */
typedef struct FwFwdCtx
{
  FwFwd* fwd;             // T,F,I,D,N
  TscTime rxTime;         // T(=now),F,I,D,N
  SgEvent eventKind;      // T,F,I,D,N
  FibNexthopFilter nhFlt; // T,I,D,N
  union
  {
    Packet* npkt;
    struct rte_mbuf* pkt;
  };                  // F,D,N
  FibEntry* fibEntry; // T,I,D,N
  PitEntry* pitEntry; // T,I,D,N

  // end of SgCtx fields
  char endofSgCtx[0];

  PitUp* pitUp;     // N
  uint64_t rxToken; // F,I,D,N
  uint32_t dnNonce; // I
  int nForwarded;   // T,I,N
  FaceID rxFace;    // F,I,D
} FwFwdCtx;

__attribute__((nonnull)) void
FwFwd_RxInterest(FwFwd* fwd, FwFwdCtx* ctx);

__attribute__((nonnull)) void
FwFwd_RxData(FwFwd* fwd, FwFwdCtx* ctx);

__attribute__((nonnull)) void
FwFwd_RxNack(FwFwd* fwd, FwFwdCtx* ctx);

#ifdef NDEBUG
#define FwFwd_NULLize(x) (void)(x)
#else
/** @brief Set x to NULL to crash on memory access bugs. */
#define FwFwd_NULLize(x)                                                                           \
  do {                                                                                             \
    (x) = NULL;                                                                                    \
  } while (false)
#endif

#endif // NDNDPDK_FWDP_FWD_H
