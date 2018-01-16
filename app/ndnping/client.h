#ifndef NDN_DPDK_APP_NDNPING_CLIENT_H
#define NDN_DPDK_APP_NDNPING_CLIENT_H

#include "../../container/nameset/nameset.h"
#include "../../iface/face.h"

/** \brief Per-pattern information in ndnping client.
 */
typedef struct NdnpingClientPattern
{
  uint64_t nInterests;
  uint64_t nData;
  uint64_t nNacks;
} NdnpingClientPattern;

/** \brief ndnping client.
 */
typedef struct NdnpingClient
{
  Face* face;
  NameSet patterns;
  struct rte_mempool* mpInterest; ///< mempool for Interests
  double interestInterval; ///< average interval between two Interests (millis)

  InterestTemplate interestTpl;
  struct
  {
    char _padding[6];
    uint8_t compT;
    uint8_t compL;
    uint64_t compV; // sequence number in native endianness
  } suffixComponent;

  uint64_t nAllocError;
} NdnpingClient;

/** \brief Initialize NdnpingClient.
 *
 *  The caller is reponsible for \p face, \p prefixes, \p mpInterest, \p interestInterval.
 *  This function initializes all other fields.
 */
void NdnpingClient_Init(NdnpingClient* client);

int NdnpingClient_Run(NdnpingClient* client);

#endif // NDN_DPDK_APP_NDNPING_CLIENT_H
