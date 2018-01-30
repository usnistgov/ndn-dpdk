#ifndef NDN_DPDK_APP_NDNPING_CLIENT_H
#define NDN_DPDK_APP_NDNPING_CLIENT_H

#include "../../container/nameset/nameset.h"
#include "../../core/running_stat/running-stat.h"
#include "../../iface/face.h"

/// \file

/** \brief Maximum number of patterns.
 *
 *  This is checked by \p NdnpingClient_EnableRtt.
 */
#define NDNPINGCLIENT_MAXPATTERNS 64

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
 *  Duration unit is (RDTSC >> NDNPING_TIMING_PRECISION).
 */
#define NDNPING_TIMING_PRECISION 16

/** \brief ndnping client.
 */
typedef struct NdnpingClient
{
  // config:
  Face* face;
  NameSet patterns;
  struct rte_mempool* mpInterest; ///< mempool for Interests
  float interestInterval; ///< average interval between two Interests (millis)

  /** \brief How often to sample RTT and latency.
   *
   *  A sample is taken every (2^sampleFreq) Interests.
   *  0xFF disables sampling.
   */
  uint8_t sampleFreq;
  /** \brief How many samples to keep.
   *
   *  \p sampleTable has (2^sampleTableSize) entries.
   *  The same entry will be reused after (2^(sampleFreq+sampleTableSize)) Interests are sent,
   *  and the previous Interest in the entry will be reported as timeout if not satisfied.
   */
  uint8_t sampleTableSize;

  // counters:
  uint64_t nAllocError;

  // internal:
  InterestTemplate interestTpl;
  struct
  {
    char _padding[6];
    uint8_t compT;
    uint8_t compL;
    uint64_t compV; ///< sequence number in native endianness
  } suffixComponent;

  _Atomic uint64_t tscHz;

  /** \brief Bitmask to determine whether to sample a packet.
   *
   *  (seqNo & samplingMask) == 0
   */
  uint64_t samplingMask;
  /** \brief Bitmask to determine where to store sample in \p sampleTable .
   *
   *  sampleTable[(seqNo >> sampleFreq) & sampleIndexMask]
   */
  uint64_t sampleIndexMask;
  void* sampleTable;
} NdnpingClient;

/** \brief Initialize NdnpingClient.
 *  \pre Config fields are initialized.
 */
void NdnpingClient_Init(NdnpingClient* client);

void NdnpingClient_Close(NdnpingClient* client);

void NdnpingClient_Run(NdnpingClient* client);

/** \brief Get RDTSC frequency (in Hz) for the lcore executing \p NdnpingClient_Run.
 */
uint64_t NdnpingClient_GetTscHz(NdnpingClient* client);

#endif // NDN_DPDK_APP_NDNPING_CLIENT_H
