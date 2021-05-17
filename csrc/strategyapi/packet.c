#include "../ndni/packet.h"
#include "packet.h"

static_assert(offsetof(SgPacket, rxFace) == offsetof(struct rte_mbuf, port), "");
static_assert(offsetof(SgPacket, packet_type_) == offsetof(struct rte_mbuf, packet_type), "");
static_assert(offsetof(SgPacket, mbuf_end_) == sizeof(struct rte_mbuf), "");

static_assert(offsetof(SgPacket, nackReason) - offsetof(SgPacket, mbuf_end_) ==
                offsetof(PacketPriv, lpl3) + offsetof(LpL3, nackReason),
              "");
static_assert(offsetof(SgPacket, congMark) - offsetof(SgPacket, mbuf_end_) ==
                offsetof(PacketPriv, lpl3) + offsetof(LpL3, congMark),
              "");

static_assert((int)SgNackCongestion == (int)NackCongestion, "");
static_assert((int)SgNackDuplicate == (int)NackDuplicate, "");
static_assert((int)SgNackNoRoute == (int)NackNoRoute, "");
static_assert((int)SgNackUnspecified == (int)NackUnspecified, "");
