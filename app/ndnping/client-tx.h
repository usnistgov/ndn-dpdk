#ifndef NDN_DPDK_APP_NDNPING_CLIENT_TX_H
#define NDN_DPDK_APP_NDNPING_CLIENT_TX_H

/// \file

#include "client-common.h"

#include "../../core/pcg_basic.h"
#include "../../dpdk/thread.h"
#include "../../dpdk/tsc.h"
#include "../../iface/face.h"
#include "../../ndn/encode-interest.h"

/** \brief Per-pattern information in ndnping client.
 */
typedef struct PingClientTxPattern
{
  uint64_t nInterests;

  struct
  {
    char _padding[6]; // make compV aligned
    uint8_t compT;
    uint8_t compL;
    uint64_t compV;      ///< sequence number in native endianness
  } __rte_packed seqNum; ///< sequence number component

  InterestTemplate tpl;
  uint8_t tplPrepareBuffer[64];
  uint8_t prefixBuffer[NAME_MAX_LENGTH];
} PingClientTxPattern;

/** \brief ndnping client.
 */
typedef struct PingClientTx
{
  FaceId face;
  uint16_t interestMbufHeadroom;
  ThreadStopFlag stop;
  uint8_t runNum;
  uint16_t nPatterns;
  struct rte_mempool* interestMp; ///< mempool for Interests
  TscDuration burstInterval;      ///< interval between two bursts

  pcg32_random_t trafficRng;
  NonceGen nonceGen;
  uint64_t nAllocError;

  PingClientTxPattern pattern[PINGCLIENT_MAX_PATTERNS];
} PingClientTx;

void
PingClientTx_Run(PingClientTx* ct);

#endif // NDN_DPDK_APP_NDNPING_CLIENT_TX_H
