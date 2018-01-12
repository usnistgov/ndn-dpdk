#ifndef NDN_DPDK_CMD_NDNPING_CLIENT_H
#define NDN_DPDK_CMD_NDNPING_CLIENT_H

#include "../../container/nameset/nameset.h"
#include "../../iface/face.h"

/** \brief ndnping client
 */
typedef struct NdnpingClient
{
  Face* face;
  NameSet prefixes;

  struct rte_mempool* mpInterest; ///< mempool for Interests

  InterestTemplate interestTpl;
  struct
  {
    char _padding[6];
    uint8_t compT;
    uint8_t compL;
    uint64_t compV; // sequence number in native endianness
  } suffixComponent;
} NdnpingClient;

void NdnpingClient_Init(NdnpingClient* client);

int NdnpingClient_Run(NdnpingClient* client);

#endif // NDN_DPDK_CMD_NDNPING_CLIENT_H
