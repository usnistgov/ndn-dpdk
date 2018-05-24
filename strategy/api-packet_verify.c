#include "api-packet.h"
#include "../ndn/packet.h"

static_assert(offsetof(SgPacket, rxFace) == offsetof(struct rte_mbuf, port),
              "");
static_assert(offsetof(SgPacket, _packet_type) ==
                offsetof(struct rte_mbuf, packet_type),
              "");
static_assert(offsetof(SgPacket, timestamp) ==
                offsetof(struct rte_mbuf, timestamp),
              "");
static_assert(offsetof(SgPacket, _mbuf_end) == sizeof(struct rte_mbuf), "");

static_assert(offsetof(SgPacket, nackReason) - offsetof(SgPacket, _mbuf_end) ==
                offsetof(PacketPriv, lpl3) + offsetof(LpL3, nackReason),
              "");
static_assert(offsetof(SgPacket, congMark) - offsetof(SgPacket, _mbuf_end) ==
                offsetof(PacketPriv, lpl3) + offsetof(LpL3, congMark),
              "");

static_assert(SgNackReason_Congestion == NackReason_Congestion, "");
static_assert(SgNackReason_Duplicate == NackReason_Duplicate, "");
static_assert(SgNackReason_NoRoute == NackReason_NoRoute, "");
static_assert(SgNackReason_Unspecified == NackReason_Unspecified, "");
