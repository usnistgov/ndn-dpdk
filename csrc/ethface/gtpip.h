#ifndef NDNDPDK_ETHFACE_GTPIP_H
#define NDNDPDK_ETHFACE_GTPIP_H

/** @file */

#include "../dpdk/hashtable.h"
#include "../dpdk/mbuf.h"

/** @brief GTP-IP handler. */
typedef struct EthGtpip {
  /**
   * @brief Mapping from UE IPv4 address to FaceID.
   *
   * In this hashtable:
   * @li Key: 4-octet IPv4 address in network byte order.
   * @li Data: 2-octet FaceID at LSB; upper bits unused.
   * @li Position: unused.
   */
  struct rte_hash* ipv4;
} EthGtpip;

/**
 * @brief Process downlink packets.
 * @param pkts Ethernet frames received on N6 interface.
 * @param count quantity of @p pkts , maximum is 64.
 * @returns bitfield of accepted packets. Use @c rte_popcount64 to obtain quantity.
 *
 * If a packet carries IP traffic that matches a known UE in @p g , its Ethernet header is removed
 * and then the packet is encapsulated in GTP-U tunnel by prepending outer Ethernet + outer IP +
 * outer UDP + GTPv1 headers.
 */
__attribute__((nonnull)) uint64_t
EthGtpip_ProcessDownlinkBulk(EthGtpip* g, struct rte_mbuf* pkts[], uint32_t count);

/**
 * @brief Process uplink packet.
 * @param m Ethernet frame received on N3 interface.
 * @returns whether the packet matches a UE.
 *
 * If @p m carries IP traffic in GTP-U tunnel that matches a known UE in @p g , returns true.
 * The outer Ethernet + outer IP + outer UDP + GTPv1 headers are removed, and then a new
 * Ethernet header is prepended.
 */
__attribute__((nonnull)) bool
EthGtpip_ProcessUplink(EthGtpip* g, struct rte_mbuf* m);

#endif // NDNDPDK_ETHFACE_GTPIP_H
