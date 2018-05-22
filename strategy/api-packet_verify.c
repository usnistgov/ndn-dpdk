#include "api-packet.h"
#include "../dpdk/mbuf.h"

static_assert(offsetof(SgPacket, rxFace) == offsetof(struct rte_mbuf, port),
              "");
static_assert(offsetof(SgPacket, _packet_type) ==
                offsetof(struct rte_mbuf, packet_type),
              "");
static_assert(offsetof(SgPacket, timestamp) ==
                offsetof(struct rte_mbuf, timestamp),
              "");
static_assert(sizeof(SgPacket) == sizeof(struct rte_mbuf), "");
