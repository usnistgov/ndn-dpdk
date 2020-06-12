#ifndef NDN_DPDK_APP_FWDP_FWD_H
#define NDN_DPDK_APP_FWDP_FWD_H

/// \file

#include "../../container/fib/fib.h"
#include "../../container/pcct/cs.h"
#include "../../container/pcct/pit.h"
#include "../../container/pktqueue/queue.h"
#include "../../core/running_stat/running-stat.h"
#include "../../dpdk/eal/thread.h"
#include "../../iface/face.h"
#include "../../strategy/api.h"

/** \brief Forwarding thread.
 */
typedef struct FwFwd
{
  SgGlobal sgGlobal;
  PktQueue inInterestQueue;
  PktQueue inDataQueue;
  PktQueue inNackQueue;

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

  struct rte_mempool* headerMp;   ///< mempool for Interest/Data header
  struct rte_mempool* guiderMp;   ///< mempool for Interest guiders
  struct rte_mempool* indirectMp; ///< mempool for indirect mbufs

  struct rte_ring* crypto; ///< queue to crypto helper

  /** \brief Statistics of latency from packet arrival to start processing.
   */
  RunningStat latencyStat;
} FwFwd;

static inline Pcct**
FwFwd_GetPcctPtr_(FwFwd* fwd)
{
  return &fwd->pcct;
}

void
FwFwd_Run(FwFwd* fwd);

/** \brief Per-packet context in forwarding.
 *
 *  Field availablility:
 *  T: set by SgTriggerTimer and available during SGEVT_TIMER
 *  F: set by FwFwd_Run
 *  I: available during SGEVT_INTEREST
 *  D: available during SGEVT_DATA
 *  N: available during SGEVT_NACK
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
  FaceId rxFace;    // F,I,D
} FwFwdCtx;

void
FwFwd_RxInterest(FwFwd* fwd, FwFwdCtx* ctx);

void
FwFwd_RxData(FwFwd* fwd, FwFwdCtx* ctx);

void
FwFwd_RxNack(FwFwd* fwd, FwFwdCtx* ctx);

#ifdef NDEBUG
#define FwFwd_NULLize(x) (void)(x)
#else
/** \brief Set x to NULL to crash on memory access bugs.
 */
#define FwFwd_NULLize(x)                                                       \
  do {                                                                         \
    (x) = NULL;                                                                \
  } while (false)
#endif

static const size_t FwFwd_OffsetofQueue[L3PktType_MAX] = {
  SIZE_MAX,
  offsetof(FwFwd, inInterestQueue),
  offsetof(FwFwd, inDataQueue),
  offsetof(FwFwd, inNackQueue),
};

#endif // NDN_DPDK_APP_FWDP_FWD_H
