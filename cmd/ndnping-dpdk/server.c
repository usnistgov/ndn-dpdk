#include "server.h"

#include "../../core/logger.h"

INIT_ZF_LOG(NdnpingServer);

#define NDNPINGSERVER_BURST_SIZE 64

static inline struct rte_mbuf*
NdnpingServer_processPkt(NdnpingServer* server, struct rte_mbuf* pkt)
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

  ZF_LOGD("%" PRI_FaceId " match-prefix=%" PRIu8, server->face->id, inNameSet);
  // TODO return Data instead of Nack-Congestion
  MakeNack(pkt, NackReason_Congestion);
  return pkt;
}

int
NdnpingServer_run(NdnpingServer* server)
{
  ZF_LOGD("%" PRI_FaceId " starting %p", server->face->id, server);
  struct rte_mbuf* pkts[NDNPINGSERVER_BURST_SIZE];
  while (true) {
    uint16_t nRx = Face_RxBurst(server->face, pkts, NDNPINGSERVER_BURST_SIZE);
    uint16_t nTx = 0;
    for (uint16_t i = 0; i < nRx; ++i) {
      pkts[nTx] = NdnpingServer_processPkt(server, pkts[i]);
      nTx += (pkts[nTx] != NULL);
    }
    Face_TxBurst(server->face, pkts, nTx);
    FreeMbufs(pkts, nTx);
  }
  return 0;
}
