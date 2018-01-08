#ifndef NDN_DPDK_CMD_NDNPING_SERVER_H
#define NDN_DPDK_CMD_NDNPING_SERVER_H

#include "../../container/nameset/nameset.h"
#include "../../iface/face.h"

typedef struct NdnpingServer
{
  Face* face;
  NameSet prefixes;     ///< served prefixes
  bool wantNackNoRoute; ///< whether to Nack unserved Interests
} NdnpingServer;

int NdnpingServer_run(NdnpingServer* server);

#endif // NDN_DPDK_CMD_NDNPING_SERVER_H
