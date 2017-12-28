#ifndef NDN_DPDK_IFACE_ETHFACE_COMMON_H
#define NDN_DPDK_IFACE_ETHFACE_COMMON_H

#include "../common.h"

#include <rte_ethdev.h>
#include <rte_ether.h>

#define _ETHFACE_LOG_PREFIX "(%" PRIu16 ",%" PRIu16 ") "
#define _ETHFACE_LOG_PARAM face->port, face->queue

#endif // NDN_DPDK_IFACE_ETHFACE_COMMON_H
