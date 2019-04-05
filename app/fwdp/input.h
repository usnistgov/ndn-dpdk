#ifndef NDN_DPDK_APP_FWDP_INPUT_H
#define NDN_DPDK_APP_FWDP_INPUT_H

/// \file

#include "fwd.h"

#include "../../container/ndt/ndt.h"
#include "../../iface/face.h"

/** \brief FwInput's connection to FwFwd.
 */
typedef struct FwInputFwdConn
{
  struct rte_ring* queue; ///< FwFwd's queue
  uint64_t nDrops;        ///< dropped packets due to full queue
} FwInputFwdConn;

/** \brief Forwarder data plane, input process.
 */
typedef struct FwInput
{
  const Ndt* ndt;
  NdtThread* ndtt;

  uint64_t nNameDisp;  ///< packets dispatched by name
  uint64_t nTokenDisp; ///< packets dispatched by token
  uint64_t nBadToken;  ///< dropped packets due to missing or bad token

  uint8_t nFwds;
  FwInputFwdConn conn[0];
} FwInput;

FwInput*
FwInput_New(const Ndt* ndt,
            uint8_t ndtThreadId,
            uint8_t nFwds,
            unsigned numaSocket);

void
FwInput_Connect(FwInput* fwi, FwFwd* fwd);

static FwInputFwdConn*
FwInput_GetConn(FwInput* fwi, uint8_t i)
{
  assert(i < fwi->nFwds);
  return &fwi->conn[i];
}

void
FwInput_DispatchByName(FwInput* fwi, Packet* npkt, const Name* name);
void
FwInput_DispatchByToken(FwInput* fwi, Packet* npkt, uint64_t token);

void
FwInput_FaceRx(FaceRxBurst* burst, void* fwi0);

#endif // NDN_DPDK_APP_FWDP_INPUT_H
