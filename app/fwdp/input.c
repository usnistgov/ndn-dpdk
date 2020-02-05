#include "input.h"

#include "token.h"

#include "../../core/logger.h"

static const size_t FwInputFwdConn_OffsetofDropCounter[L3PktType_MAX] = {
  SIZE_MAX,
  offsetof(FwInputFwdConn, nInterestDrops),
  offsetof(FwInputFwdConn, nDataDrops),
  offsetof(FwInputFwdConn, nNackDrops),
};

INIT_ZF_LOG(FwInput);

FwInput*
FwInput_New(const Ndt* ndt,
            uint8_t ndtThreadId,
            uint8_t nFwds,
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
  FwInputFwdConn* conn = FwInput_GetConn(fwi, fwd->id);
  assert(conn->fwd == NULL);
  conn->fwd = fwd;
}

static void
FwInput_PassTo(FwInput* fwi, Packet* npkt, uint8_t fwdId)
{
  FwInputFwdConn* conn = FwInput_GetConn(fwi, fwdId);
  L3PktType l3type = Packet_GetL3PktType(npkt);
  PktQueue* q = RTE_PTR_ADD(conn->fwd, FwFwd_OffsetofQueue[l3type]);
  uint32_t nRej = PktQueue_PushPlain(q, (struct rte_mbuf**)&npkt, 1);
  if (unlikely(nRej != 0)) {
    uint64_t* nDrops =
      RTE_PTR_ADD(conn, FwInputFwdConn_OffsetofDropCounter[l3type]);
    *nDrops += nRej;
  }
}

void
FwInput_DispatchByName(FwInput* fwi, Packet* npkt, const Name* name)
{
  struct rte_mbuf* pkt = Packet_ToMbuf(npkt);
  uint8_t fwdId = Ndtt_Lookup(fwi->ndt, fwi->ndtt, name);
  assert(fwdId < fwi->nFwds);

  ZF_LOGD("%s-from=%" PRI_FaceId " npkt=%p token=%016" PRIx64
          " ndt-fwd=%" PRIu8,
          L3PktType_ToString(Packet_GetL3PktType(npkt)),
          pkt->port,
          npkt,
          Packet_GetLpL3Hdr(npkt)->pitToken,
          fwdId);
  ++fwi->nNameDisp;
  FwInput_PassTo(fwi, npkt, fwdId);
}

void
FwInput_DispatchByToken(FwInput* fwi, Packet* npkt)
{
  struct rte_mbuf* pkt = Packet_ToMbuf(npkt);
  uint64_t token = Packet_GetLpL3Hdr(npkt)->pitToken;

  if (unlikely(token == 0)) {
    ZF_LOGD("%s-from=%" PRI_FaceId " npkt=%p bad-token=%016" PRIx64,
            L3PktType_ToString(Packet_GetL3PktType(npkt)),
            pkt->port,
            npkt,
            token);
    ++fwi->nBadToken;
    rte_pktmbuf_free(pkt);
    return;
  }

  uint8_t fwdId = FwToken_GetFwdId(token);

  if (unlikely(fwdId >= fwi->nFwds)) {
    ZF_LOGD("%s-from=%" PRI_FaceId " npkt=%p bad-token=%016" PRIx64,
            L3PktType_ToString(Packet_GetL3PktType(npkt)),
            pkt->port,
            npkt,
            token);
    ++fwi->nBadToken;
    rte_pktmbuf_free(pkt);
  } else {
    ZF_LOGD("%s-from=%" PRI_FaceId " npkt=%p token=%016" PRIx64
            " token-fwd=%" PRIu8,
            L3PktType_ToString(Packet_GetL3PktType(npkt)),
            pkt->port,
            npkt,
            token,
            fwdId);
    ++fwi->nTokenDisp;
    FwInput_PassTo(fwi, npkt, fwdId);
  }
}

void
FwInput_FaceRx(FaceRxBurst* burst, void* fwi0)
{
  FwInput* fwi = (FwInput*)fwi0;
  ZF_LOGD("fwi=%p burst=(%" PRIu16 "I %" PRIu16 "D %" PRIu16 "N)",
          fwi,
          burst->nInterests,
          burst->nData,
          burst->nNacks);
  for (uint16_t i = 0; i < burst->nInterests; ++i) {
    Packet* npkt = FaceRxBurst_GetInterest(burst, i);
    PInterest* interest = Packet_GetInterestHdr(npkt);
    FwInput_DispatchByName(fwi, npkt, &interest->name);
  }
  for (uint16_t i = 0; i < burst->nData; ++i) {
    Packet* npkt = FaceRxBurst_GetData(burst, i);
    FwInput_DispatchByToken(fwi, npkt);
  }
  for (uint16_t i = 0; i < burst->nNacks; ++i) {
    Packet* npkt = FaceRxBurst_GetNack(burst, i);
    FwInput_DispatchByToken(fwi, npkt);
  }
}
