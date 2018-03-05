#include "server.h"

#include "../../core/logger.h"
#include "../../ndn/encode-data.h"

INIT_ZF_LOG(NdnpingServer);

#define NDNPINGSERVER_BURST_SIZE 64

static Packet*
NdnpingServer_MakeData(NdnpingServer* server, LName name)
{
  struct rte_mbuf* m1 = rte_pktmbuf_alloc(server->data1Mp);
  if (unlikely(m1 == NULL)) {
    ZF_LOGW("%" PRI_FaceId " data1Mp is full", server->face->id);
    return NULL;
  }
  struct rte_mbuf* m2 = rte_pktmbuf_alloc(server->data2Mp);
  if (unlikely(m2 == NULL)) {
    ZF_LOGW("%" PRI_FaceId " data2Mp is full", server->face->id);
    rte_pktmbuf_free(m1);
    return NULL;
  }
  struct rte_mbuf* payload =
    rte_pktmbuf_clone(server->payload, server->indirectMp);
  if (unlikely(payload == NULL)) {
    ZF_LOGW("%" PRI_FaceId " indirectMp is full", server->face->id);
    rte_pktmbuf_free(m1);
    rte_pktmbuf_free(m2);
    return NULL;
  }

  m1->data_off = EncodeData1_GetHeadroom();
  m2->data_off = EncodeData2_GetHeadroom();

  EncodeData1(m1, name, payload);
  EncodeData2(m2, m1);
  EncodeData3(m1);

  Packet* npkt = Packet_FromMbuf(m1);
  Packet_SetL2PktType(npkt, L2PktType_None);
  Packet_SetL3PktType(npkt, L3PktType_Data); // for stats; PData* is not filled
  return npkt;
}

static Packet*
NdnpingServer_ProcessPkt(NdnpingServer* server, Packet* npkt)
{
  assert(Packet_GetL3PktType(npkt) == L3PktType_Interest);
  struct rte_mbuf* pkt = Packet_ToMbuf(npkt);

  const LName name = *(const LName*)&Packet_GetInterestHdr(npkt)->name;

  int patternId = NameSet_FindPrefix(&server->patterns, name);
  if (patternId < 0) {
    ZF_LOGV("%" PRI_FaceId " no-prefix-match", server->face->id);
    ++server->nNoMatch;
    MakeNack(npkt, NackReason_NoRoute);
    return npkt;
  }
  ZF_LOGV("%" PRI_FaceId " match-pattern=%d", server->face->id, patternId);

  NdnpingServerPattern* pattern =
    NameSet_GetUsrT(&server->patterns, patternId, NdnpingServerPattern*);
  ++pattern->nInterests;

  Packet* dataPkt = NdnpingServer_MakeData(server, name);
  if (unlikely(dataPkt == NULL)) {
    ++server->nAllocError;
    MakeNack(npkt, NackReason_Congestion);
    return npkt;
  }

  rte_pktmbuf_free(pkt);
  return dataPkt;
}

void
NdnpingServer_Rx(Face* face, FaceRxBurst* burst, void* server0)
{
  NdnpingServer* server = (NdnpingServer*)server0;
  FreeMbufs((struct rte_mbuf**)FaceRxBurst_ListData(burst), burst->nData);
  FreeMbufs((struct rte_mbuf**)FaceRxBurst_ListNacks(burst), burst->nNacks);

  Packet** tx = FaceRxBurst_ListData(burst);
  uint16_t nTx = 0;
  for (uint16_t i = 0; i < burst->nInterests; ++i) {
    tx[nTx] =
      NdnpingServer_ProcessPkt(server, FaceRxBurst_GetInterest(burst, i));
    nTx += (tx[nTx] != NULL);
  }
  Face_TxBurst(server->face, tx, nTx);
}
