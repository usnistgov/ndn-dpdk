#ifndef NDN_DPDK_CMD_NDNPING_SERVER_H
#define NDN_DPDK_CMD_NDNPING_SERVER_H

#include "../../iface/face.h"

typedef struct NdnpingServer
{
  Face* face;
} NdnpingServer;

int NdnpingServer_run(NdnpingServer* server);

#endif // NDN_DPDK_CMD_NDNPING_SERVER_H
