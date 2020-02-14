#ifndef NDN_DPDK_APP_PINGCLIENT_TX_H
#define NDN_DPDK_APP_PINGCLIENT_TX_H

/// \file

#include "common.h"

#include "../../core/pcg_basic.h"
#include "../../dpdk/thread.h"
#include "../../dpdk/tsc.h"
#include "../../iface/face.h"

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

  uint32_t seqNumOffset;

  InterestTemplate tpl;
} PingClientTxPattern;

static_assert(offsetof(PingClientTxPattern, seqNum.compV) % sizeof(uint64_t) ==
                0,
              "");

/** \brief ndnping client.
 */
typedef struct PingClientTx
{
  FaceId face;
  ThreadStopFlag stop;
  uint8_t runNum;
  uint16_t nWeights;
  struct rte_mempool* interestMp; ///< mempool for Interests
  TscDuration burstInterval;      ///< interval between two bursts

  pcg32_random_t trafficRng;
  NonceGen nonceGen;
  uint64_t nAllocError;

  PingPatternId weight[PINGCLIENT_MAX_SUM_WEIGHT];
  PingClientTxPattern pattern[PINGCLIENT_MAX_PATTERNS];
} PingClientTx;

void
PingClientTx_Run(PingClientTx* ct);

#endif // NDN_DPDK_APP_PINGCLIENT_TX_H
