#ifndef NDN_DPDK_APP_NDNPING_SERVER_H
#define NDN_DPDK_APP_NDNPING_SERVER_H

/// \file

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
  NameSet patterns;     ///< served prefixes
  bool wantNackNoRoute; ///< whether to Nack unserved Interests
  uint32_t freshnessPeriod;

  struct rte_mempool* dataMp; ///< mempool for Data
  uint16_t dataMbufHeadroom;

  uint64_t nNoMatch;
  uint64_t nAllocError;
} NdnpingServer;

void NdnpingServer_Rx(Face* face, FaceRxBurst* burst, void* server0);

#endif // NDN_DPDK_APP_NDNPING_SERVER_H
