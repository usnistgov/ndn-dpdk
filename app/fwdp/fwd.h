#ifndef NDN_DPDK_APP_FWDP_FWD_H
#define NDN_DPDK_APP_FWDP_FWD_H

#include "../../container/fib/fib.h"
#include "../../container/pcct/cs.h"
#include "../../container/pcct/pit.h"
#include "../../iface/facetable.h"

/** \brief Forwarder data plane, forwarding process.
 */
typedef struct FwFwd
{
  struct rte_ring* queue; ///< input queue

  FaceTable* ft;
  Fib* fib;
  Pit* pit;
  Cs* cs;

  uint8_t id; ///< fwd process id
  bool stop;  ///< set to true to stop the process
} FwFwd;

void FwFwd_Run(FwFwd* fwd);

void FwFwd_RxInterest(FwFwd* fwd, Packet* npkt);

void FwFwd_RxData(FwFwd* fwd, Packet* npkt);

void FwFwd_RxNack(FwFwd* fwd, Packet* npkt);

#endif // NDN_DPDK_APP_FWDP_FWD_H
