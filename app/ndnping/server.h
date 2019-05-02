#ifndef NDN_DPDK_APP_NDNPING_SERVER_H
#define NDN_DPDK_APP_NDNPING_SERVER_H

/// \file

#include "../../dpdk/thread.h"
#include "../../iface/face.h"

#define PINGSERVER_MAX_PATTERNS 256
#define PINGSERVER_BURST_SIZE 64
#define PINGSERVER_PAYLOAD_MAX 65536

/** \brief Per-pattern information in ndnping server.
 */
typedef struct PingServerPattern
{
  uint64_t nInterests;
  LName prefix;
  LName suffix;
  uint16_t payloadL;
  uint32_t freshnessPeriod;
  char nameBuffer[NAME_MAX_LENGTH];
} PingServerPattern;

/** \brief ndnping server.
 */
typedef struct PingServer
{
  struct rte_ring* rxQueue;
  struct rte_mempool* dataMp; ///< mempool for Data
  uint16_t dataMbufHeadroom;
  FaceId face;
  uint16_t nPatterns;
  bool wantNackNoRoute; ///< whether to Nack unserved Interests

  ThreadStopFlag stop;
  uint64_t nNoMatch;
  uint64_t nAllocError;

  PingServerPattern pattern[PINGSERVER_MAX_PATTERNS];
} PingServer;

void
PingServer_Run(PingServer* server);

#endif // NDN_DPDK_APP_NDNPING_SERVER_H
