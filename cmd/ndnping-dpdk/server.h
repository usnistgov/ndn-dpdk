#ifndef NDN_DPDK_CMD_NDNPING_SERVER_H
#define NDN_DPDK_CMD_NDNPING_SERVER_H

#include "../../container/nameset/nameset.h"
#include "../../iface/face.h"

typedef struct NdnpingServer
{
  Face* face;
  NameSet prefixes;         ///< served prefixes
  bool wantNackNoRoute;     ///< whether to Nack unserved Interests
  struct rte_mbuf* payload; ///< the payload

  struct rte_mempool* mpData1;    ///< mempool for Data header
  struct rte_mempool* mpData2;    ///< mempool for Data signature
  struct rte_mempool* mpIndirect; ///< mempool for indirect mbufs to payload
} NdnpingServer;

int NdnpingServer_Run(NdnpingServer* server);

#endif // NDN_DPDK_CMD_NDNPING_SERVER_H
