#ifndef NDN_DPDK_APP_NDNPING_SERVER_H
#define NDN_DPDK_APP_NDNPING_SERVER_H

#include "../../container/nameset/nameset.h"
#include "../../iface/face.h"

/** \brief Per-pattern information in ndnping server.
 */
typedef struct NdnpingServerPattern
{
  uint64_t nInterests;
} NdnpingServerPattern;

/** \brief ndnping server.
 */
typedef struct NdnpingServer
{
  Face* face;
  NameSet patterns;         ///< served prefixes
  bool wantNackNoRoute;     ///< whether to Nack unserved Interests
  struct rte_mbuf* payload; ///< the payload

  struct rte_mempool* mpData1;    ///< mempool for Data header
  struct rte_mempool* mpData2;    ///< mempool for Data signature
  struct rte_mempool* mpIndirect; ///< mempool for indirect mbufs to payload

  uint64_t nNoMatch;
  uint64_t nAllocError;
} NdnpingServer;

void NdnpingServer_Run(NdnpingServer* server);

#endif // NDN_DPDK_APP_NDNPING_SERVER_H
