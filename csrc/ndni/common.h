#ifndef NDN_DPDK_NDNI_COMMON_H
#define NDN_DPDK_NDNI_COMMON_H

/** @file */

#include "../core/common.h"
#include <rte_byteorder.h>

#include "../dpdk/cryptodev.h"
#include "../dpdk/mbuf.h"

#include "an.h"
#include "enum.h"

typedef struct Packet Packet;
typedef struct PInterest PInterest;
typedef struct PData PData;
typedef struct PNack PNack;

#endif // NDN_DPDK_NDNI_COMMON_H
