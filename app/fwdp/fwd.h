#ifndef NDN_DPDK_APP_FWDP_FWD_H
#define NDN_DPDK_APP_FWDP_FWD_H

/// \file

#include "../../container/fib/fib.h"
#include "../../container/pcct/cs.h"
#include "../../container/pcct/pit.h"
#include "../../core/running_stat/running-stat.h"
#include "../../iface/facetable.h"

/** \brief Forwarder data plane, forwarding process.
 */
typedef struct FwFwd
{
  struct rte_ring* queue; ///< input queue

  FaceTable* ft;
  Fib* fib;
  union
  {
    Pcct* pcct;
    Pit* pit;
    Cs* cs;
  };

  uint8_t id; ///< fwd process id
  bool stop;  ///< set to true to stop the process

  struct rte_mempool* headerMp;   ///< mempool for cloned packet header
  struct rte_mempool* indirectMp; ///< mempool for indirect mbufs

  /** \brief Statistics of latency from packet arrival to start processing.
   */
  RunningStat timeSinceRxStat;
} FwFwd;

static Pcct**
__FwFwd_GetPcctPtr(FwFwd* fwd)
{
  return &fwd->pcct;
}

void FwFwd_Run(FwFwd* fwd);

void FwFwd_RxInterest(FwFwd* fwd, Packet* npkt);

void FwFwd_RxData(FwFwd* fwd, Packet* npkt);

void FwFwd_RxNack(FwFwd* fwd, Packet* npkt);

#endif // NDN_DPDK_APP_FWDP_FWD_H
