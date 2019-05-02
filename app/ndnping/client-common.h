#ifndef NDN_DPDK_APP_NDNPING_CLIENT_COMMON_H
#define NDN_DPDK_APP_NDNPING_CLIENT_COMMON_H

/// \file

#define PINGCLIENT_MAX_PATTERNS 256
#define PINGCLIENT_RX_BURST_SIZE 64
#define PINGCLIENT_TX_BURST_SIZE 64

#define PINGCLIENT_SUFFIX_LEN 10 // T+L+sizeof(uint64)

#define PINGCLIENT_SELECT_PATTERN(client, seqNum)                              \
  ((seqNum) % (client)->nPatterns)

#endif // NDN_DPDK_APP_NDNPING_CLIENT_COMMON_H
