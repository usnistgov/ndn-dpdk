#include "../ndni/packet.h"
#include "packet.h"

static_assert(offsetof(SgPacket, rxFace) == offsetof(struct rte_mbuf, port), "");
static_assert(offsetof(SgPacket, packet_type_) == offsetof(struct rte_mbuf, packet_type), "");
static_assert(offsetof(SgPacket, endofMbuf_) == sizeof(struct rte_mbuf), "");

static_assert(offsetof(SgPacket, nackReason) - offsetof(SgPacket, endofMbuf_) ==
                offsetof(PacketPriv, lpl3.nackReason),
              "");
static_assert(offsetof(SgPacket, congMark) - offsetof(SgPacket, endofMbuf_) ==
                offsetof(PacketPriv, lpl3.congMark),
              "");
