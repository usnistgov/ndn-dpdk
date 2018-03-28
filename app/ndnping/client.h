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

/** \brief Precision of timing measurements.
 *
 *  Duration unit is (TSC >> NDNPING_TIMING_PRECISION).
 */
#define NDNPING_TIMING_PRECISION 16

/** \brief ndnping client.
 */
typedef struct NdnpingClient
{
  // basic config:
  Face* face;
  NameSet patterns;
  struct rte_mempool* interestMp; ///< mempool for Interests
  uint16_t interestMbufHeadroom;
  float interestInterval; ///< average interval between two Interests (millis)

  // sampling config:
  /** \brief How often to sample RTT and latency.
   *
   *  A sample is taken every (2^sampleFreq) Interests.
   */
  uint8_t sampleFreq;
  /** \brief How many samples to keep.
   *
   *  \c sampleTable has (2^sampleTableSize) entries.
   *  The same entry will be reused after (2^(sampleFreq+sampleTableSize)) Interests are sent,
   *  and the previous Interest in the entry will be reported as timeout if not satisfied.
   */
  uint8_t sampleTableSize;

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

  /** \brief Bitmask to determine whether to sample a packet.
   *
   *  (seqNo & samplingMask) == 0
   */
  uint64_t samplingMask;
  /** \brief Bitmask to determine where to store sample in \c sampleTable .
   *
   *  sampleTable[(seqNo >> sampleFreq) & sampleIndexMask]
   */
  uint64_t sampleIndexMask;
  void* sampleTable;

  uint8_t interestPrepareBuffer[8192];
} NdnpingClient;

/** \brief Initialize NdnpingClient.
 *  \pre Basic config fields are initialized.
 */
void NdnpingClient_Init(NdnpingClient* client);

/** \brief Initialize NdnpingClient.
 *  \pre Sampling config fields are initialized.
 */
void NdnpingClient_EnableSampling(NdnpingClient* client);

void NdnpingClient_Close(NdnpingClient* client);

void NdnpingClient_RunTx(NdnpingClient* client);

void NdnpingClient_Rx(Face* face, FaceRxBurst* burst, void* client0);

#endif // NDN_DPDK_APP_NDNPING_CLIENT_H
