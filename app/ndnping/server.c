#include "server.h"

#include "../../core/logger.h"
#include "../../ndn/encode-data.h"

INIT_ZF_LOG(NdnpingServer);

#define NDNPINGSERVER_BURST_SIZE 64

static Packet*
NdnpingServer_MakeData(NdnpingServer* server, LName name)
{
  struct rte_mbuf* m = rte_pktmbuf_alloc(server->dataMp);
  if (unlikely(m == NULL)) {
    ZF_LOGW("dataMp-full");
    return NULL;
  }
  m->data_off = server->dataMbufHeadroom;
  EncodeData(m, name, server->nameSuffix, server->freshnessPeriod,
             server->payloadL, server->payloadV);

  Packet* npkt = Packet_FromMbuf(m);
  Packet_SetL2PktType(npkt, L2PktType_None);
  Packet_SetL3PktType(npkt, L3PktType_Data); // for stats; no PData*
  return npkt;
}

static Packet*
NdnpingServer_ProcessPkt(NdnpingServer* server, Packet* npkt)
{
  assert(Packet_GetL3PktType(npkt) == L3PktType_Interest);
  struct rte_mbuf* pkt = Packet_ToMbuf(npkt);
  uint64_t token = Packet_GetLpL3Hdr(npkt)->pitToken;

  const LName name = *(const LName*)&Packet_GetInterestHdr(npkt)->name;

  int patternId = NameSet_FindPrefix(&server->patterns, name);
  if (patternId < 0) {
    ZF_LOGD(">I dn-token=%016" PRIx64 " no-pattern", token);
    ++server->nNoMatch;
    if (server->wantNackNoRoute) {
      MakeNack(npkt, NackReason_NoRoute);
      return npkt;
    } else {
      rte_pktmbuf_free(pkt);
      return NULL;
    }
  }
  ZF_LOGD(">I dn-token=%016" PRIx64 " pattern=%d", token, patternId);

  NdnpingServerPattern* pattern =
    NameSet_GetUsrT(&server->patterns, patternId, NdnpingServerPattern*);
  ++pattern->nInterests;

  Packet* dataPkt = NdnpingServer_MakeData(server, name);
  if (unlikely(dataPkt == NULL)) {
    ++server->nAllocError;
    MakeNack(npkt, NackReason_Congestion);
    return npkt;
  }

  Packet_InitLpL3Hdr(dataPkt)->pitToken = token;
  rte_pktmbuf_free(pkt);
  return dataPkt;
}

void
NdnpingServer_Rx(FaceId faceId, FaceRxBurst* burst, void* server0)
{
  NdnpingServer* server = (NdnpingServer*)server0;
  ZF_LOGD("server-face=%" PRI_FaceId " burst=(%" PRIu16 "I %" PRIu16
          "D %" PRIu16 "N)",
          server->face, burst->nInterests, burst->nData, burst->nNacks);
  FreeMbufs((struct rte_mbuf**)FaceRxBurst_ListData(burst), burst->nData);
  FreeMbufs((struct rte_mbuf**)FaceRxBurst_ListNacks(burst), burst->nNacks);

  Packet** tx = FaceRxBurst_ListData(burst);
  uint16_t nTx = 0;
  for (uint16_t i = 0; i < burst->nInterests; ++i) {
    Packet* npkt = FaceRxBurst_GetInterest(burst, i);
    tx[nTx] = NdnpingServer_ProcessPkt(server, npkt);
    nTx += (tx[nTx] != NULL);
  }
  Face_TxBurst(server->face, tx, nTx);
}
