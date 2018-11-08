#ifndef NDN_DPDK_APP_NDNPING_CLIENT_H
#define NDN_DPDK_APP_NDNPING_CLIENT_H

/// \file

#include "../../container/nameset/nameset.h"
#include "../../core/running_stat/running-stat.h"
#include "../../dpdk/tsc.h"
#include "../../iface/face.h"
#include "../../ndn/encode-interest.h"

/** \brief Per-pattern information in ndnping client.
 */
typedef struct NdnpingClientPattern
{
  uint64_t nInterests;
  uint64_t nData;
  uint64_t nNacks;

  RunningStat rtt;
} NdnpingClientPattern;

/** \brief ndnping client.
 */
typedef struct NdnpingClient
{
  // basic config:
  struct rte_ring* rxQueue;
  FaceId face;

  NameSet patterns;
  struct rte_mempool* interestMp; ///< mempool for Interests
  uint16_t interestMbufHeadroom;
  float interestInterval; ///< average interval between two Interests (millis)

  // counters:
  uint64_t nAllocError;

  // private:
  InterestTemplate interestTpl;
  struct
  {
    char _padding[6];
    uint8_t compT;
    uint8_t compL;
    uint64_t compV; ///< sequence number in native endianness
  } __rte_packed suffixComponent;
  NonceGen nonceGen;

  uint8_t interestPrepareBuffer[8192];
} NdnpingClient;

/** \brief Initialize NdnpingClient.
 *  \pre Basic config fields are initialized.
 */
void NdnpingClient_Init(NdnpingClient* client);

void NdnpingClient_Close(NdnpingClient* client);

void NdnpingClient_RunTx(NdnpingClient* client);

void NdnpingClient_RunRx(NdnpingClient* client);

#endif // NDN_DPDK_APP_NDNPING_CLIENT_H
