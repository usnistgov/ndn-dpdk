#include "input.h"

#include "token.h"

#include "../../core/logger.h"

INIT_ZF_LOG(FwInput);

FwInput*
FwInput_New(const Ndt* ndt, uint8_t ndtThreadId, uint8_t nFwds,
            unsigned numaSocket)
{
  size_t size = sizeof(FwInput) + sizeof(FwInputFwdConn) * nFwds;
  FwInput* fwi = (FwInput*)rte_zmalloc_socket("FwInput", size, 0, numaSocket);
  fwi->ndt = ndt;
  fwi->ndtt = Ndt_GetThread(ndt, ndtThreadId);
  fwi->nFwds = nFwds;
  return fwi;
}

void
FwInput_Connect(FwInput* fwi, FwFwd* fwd)
{
  FwInputFwdConn* conn = &fwi->conn[fwd->id];
  assert(conn->queue == NULL);
  conn->queue = fwd->queue;
}

static void
FwInput_PassTo(FwInput* fwi, Packet* npkt, uint8_t fwdId)
{
  int res = rte_ring_enqueue(fwi->conn[fwdId].queue, npkt);
  if (res != 0) {
    ++fwi->conn[fwdId].nDrops;
    rte_pktmbuf_free(Packet_ToMbuf(npkt));
  }
}

static void
FwInput_DispatchByName(FwInput* fwi, Packet* npkt, const Name* name)
{
  uint8_t fwdId = Ndt_Lookup(fwi->ndt, fwi->ndtt, &name->p, name->v);
  ZF_LOGD("%" PRI_FaceId " %p by-name to=%" PRIu8, Packet_ToMbuf(npkt)->port,
          npkt, fwdId);
  FwInput_PassTo(fwi, npkt, fwdId);
}

static void
FwInput_DispatchByToken(FwInput* fwi, Packet* npkt, uint64_t token)
{
  FwToken tok = {.token = token };
  uint8_t fwdId = tok.fwdId;

  if (unlikely(fwdId >= fwi->nFwds)) {
    ZF_LOGD("%" PRI_FaceId " %p token=%" PRIx64 " bad-fwdId",
            Packet_ToMbuf(npkt)->port, npkt, token);
    rte_pktmbuf_free(Packet_ToMbuf(npkt));
  } else {
    ZF_LOGD("%" PRI_FaceId " %p token=%" PRIx64 " to=%" PRIu8,
            Packet_ToMbuf(npkt)->port, npkt, token, fwdId);
    FwInput_PassTo(fwi, npkt, fwdId);
  }
}

void
FwInput_FaceRx(Face* face, FaceRxBurst* burst, void* fwi0)
{
  FwInput* fwi = (FwInput*)fwi0;
  for (uint16_t i = 0; i < burst->nInterests; ++i) {
    Packet* npkt = FaceRxBurst_GetInterest(burst, i);
    PInterest* interest = Packet_GetInterestHdr(npkt);
    FwInput_DispatchByName(fwi, npkt, &interest->name);
  }
  for (uint16_t i = 0; i < burst->nData; ++i) {
    Packet* npkt = FaceRxBurst_GetData(burst, i);
    LpL3* lpl3 = Packet_GetLpL3Hdr(npkt);
    FwInput_DispatchByToken(fwi, npkt, lpl3->pitToken);
  }
  for (uint16_t i = 0; i < burst->nNacks; ++i) {
    Packet* npkt = FaceRxBurst_GetNack(burst, i);
    LpL3* lpl3 = Packet_GetLpL3Hdr(npkt);
    FwInput_DispatchByToken(fwi, npkt, lpl3->pitToken);
  }
}
