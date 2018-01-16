#include "server.h"

#include "../../core/logger.h"

INIT_ZF_LOG(NdnpingServer);

#define NDNPINGSERVER_BURST_SIZE 64

static inline struct rte_mbuf*
NdnpingServer_MakeData(NdnpingServer* server, const Name* name)
{
  struct rte_mbuf* m1 = rte_pktmbuf_alloc(server->mpData1);
  if (m1 == NULL) {
    ZF_LOGW("%" PRI_FaceId " mpData1 is full", server->face->id);
    return NULL;
  }
  struct rte_mbuf* m2 = rte_pktmbuf_alloc(server->mpData2);
  if (m2 == NULL) {
    ZF_LOGW("%" PRI_FaceId " mpData2 is full", server->face->id);
    rte_pktmbuf_free(m1);
    return NULL;
  }
  struct rte_mbuf* payload =
    rte_pktmbuf_clone(server->payload, server->mpIndirect);
  if (payload == NULL) {
    ZF_LOGW("%" PRI_FaceId " mpIndirect is full", server->face->id);
    rte_pktmbuf_free(m1);
    rte_pktmbuf_free(m2);
    return NULL;
  }

  m1->data_off = EncodeData1_GetHeadroom();
  m2->data_off = EncodeData2_GetHeadroom();

  EncodeData1(m1, name, payload);
  EncodeData2(m2, m1);
  EncodeData3(m1);
  return m1;
}

static inline struct rte_mbuf*
NdnpingServer_ProcessPkt(NdnpingServer* server, struct rte_mbuf* pkt)
{
  if (Packet_GetNdnPktType(pkt) != NdnPktType_Interest) {
    ZF_LOGD("%" PRI_FaceId " not-Interest", server->face->id);
    rte_pktmbuf_free(pkt);
    return NULL;
  }

  Name* name = &Packet_GetInterestHdr(pkt)->name;
  const uint8_t* nameComps = Name_LinearizeComps(name, 0);

  int inNameSet =
    NameSet_FindPrefix(&server->prefixes, nameComps, name->nOctets);
  if (inNameSet < 0) {
    ZF_LOGD("%" PRI_FaceId " no-prefix-match", server->face->id);
    MakeNack(pkt, NackReason_NoRoute);
    return pkt;
  }
  ZF_LOGV("%" PRI_FaceId " match-prefix=%" PRIu8, server->face->id, inNameSet);

  struct rte_mbuf* dataPkt = NdnpingServer_MakeData(server, name);
  if (unlikely(dataPkt == NULL)) {
    MakeNack(pkt, NackReason_Congestion);
  }

  rte_pktmbuf_free(pkt);
  return dataPkt;
}

void
NdnpingServer_Run(NdnpingServer* server)
{
  ZF_LOGD("%" PRI_FaceId " starting %p", server->face->id, server);
  struct rte_mbuf* pkts[NDNPINGSERVER_BURST_SIZE];
  while (true) {
    uint16_t nRx = Face_RxBurst(server->face, pkts, NDNPINGSERVER_BURST_SIZE);
    uint16_t nTx = 0;
    for (uint16_t i = 0; i < nRx; ++i) {
      pkts[nTx] = NdnpingServer_ProcessPkt(server, pkts[i]);
      nTx += (pkts[nTx] != NULL);
    }
    Face_TxBurst(server->face, pkts, nTx);
    FreeMbufs(pkts, nTx);
  }
}
