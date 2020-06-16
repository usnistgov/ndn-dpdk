#include "packet.h"
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

static_assert((int)SgNackCongestion == (int)NackCongestion, "");
static_assert((int)SgNackDuplicate == (int)NackDuplicate, "");
static_assert((int)SgNackNoRoute == (int)NackNoRoute, "");
static_assert((int)SgNackUnspecified == (int)NackUnspecified, "");
