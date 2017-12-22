#ifndef NDN_DPDK_NDNFACE_COMMON_H
#define NDN_DPDK_NDNFACE_COMMON_H

#include "../core/common.h"

#include <rte_ethdev.h>
#include <rte_ether.h>

#include "../ndn/nack-pkt.h"
#include "../ndn/packet.h"
#include "../ndn/protonum.h"

#define _NDNFACE_LOG_PREFIX "(%" PRIu16 ",%" PRIu16 ") "
#define _NDNFACE_LOG_PARAM face->port, face->queue

#endif // NDN_DPDK_NDNFACE_COMMON_H
