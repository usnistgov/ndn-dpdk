#include "server.h"

#define NDNPINGSERVER_BURST_SIZE 64

int
NdnpingServer_run(NdnpingServer* server)
{
  struct rte_mbuf* pkts[NDNPINGSERVER_BURST_SIZE];
  while (true) {
    uint16_t nPkts = Face_RxBurst(server->face, pkts, NDNPINGSERVER_BURST_SIZE);
    for (uint16_t i = 0; i < nPkts; ++i) {
      struct rte_mbuf* pkt = pkts[i];
      MakeNack(pkt, NackReason_Congestion);
    }
    Face_TxBurst(server->face, pkts, nPkts);
    FreeMbufs(pkts, nPkts);
  }
  return 0;
}
