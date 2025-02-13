#include "locator.h"
#include "hdr-impl.h"

EthLocatorClass
EthLocator_Classify(const EthLocator* loc) {
  EthLocatorClass c = {0};
  if (rte_is_zero_ether_addr(&loc->local)) {
    if (rte_is_broadcast_ether_addr(&loc->remote)) {
      c.passthru = true;
    }
    return c;
  }
  c.multicast = rte_is_multicast_ether_addr(&loc->remote);
  c.udp = loc->remoteUDP != 0;
  // as of DPDK 24.11, rte_ipv6_addr_is_v4mapped has a bug:
  // it is passing depth=32 to rte_ipv6_addr_eq_prefix, should be depth=96
  const struct rte_ipv6_addr v4mappedPrefix = RTE_IPV6_ADDR_PREFIX_V4MAPPED;
  c.v4 =
    rte_ipv6_addr_eq_prefix(&loc->remoteIP, &v4mappedPrefix, V4_IN_V6_PREFIX_OCTETS * CHAR_BIT);
  c.tunnel = 0;
  if (!rte_is_zero_ether_addr(&loc->innerRemote)) {
    c.tunnel = 'V';
  } else if (loc->isGtp) {
    c.tunnel = 'G';
  }
  c.etherType = !c.udp ? EtherTypeNDN : c.v4 ? RTE_ETHER_TYPE_IPV4 : RTE_ETHER_TYPE_IPV6;
  return c;
}

bool
EthLocator_CanCoexist(const EthLocator* a, const EthLocator* b) {
  EthLocatorClass ac = EthLocator_Classify(a);
  EthLocatorClass bc = EthLocator_Classify(b);
  if ((ac.etherType == 0 && !ac.passthru) || (bc.etherType == 0 && !bc.passthru)) {
    // only one memif face allowed
    return false;
  }
  if (ac.passthru || bc.passthru) {
    // only one passthru face allowed
    // passthru and non-passthru can coexist
    return ac.passthru != bc.passthru;
  }
  if (ac.multicast != bc.multicast || ac.udp != bc.udp || ac.v4 != bc.v4) {
    // Ethernet unicast and multicast can coexist
    // Ethernet, IPv4-UDP, and IPv6-UDP can coexist
    return true;
  }
  if (ac.multicast) {
    // only one Ethernet multicast face allowed
    return false;
  }
  if (a->vlan != b->vlan) {
    // different VLAN can coexist
    return true;
  }
  if (!ac.udp) {
    if (rte_is_same_ether_addr(&a->local, &b->local) &&
        rte_is_same_ether_addr(&a->remote, &b->remote)) {
      // Ethernet faces with same MAC addresses and VLAN conflict
      return false;
    }
    // Ethernet faces with different unicast MAC addresses can coexist
    return true;
  }
  if (!rte_ipv6_addr_eq(&a->localIP, &b->localIP) ||
      !rte_ipv6_addr_eq(&a->remoteIP, &b->remoteIP)) {
    // different IP addresses can coexist
    return true;
  }
  if (ac.tunnel == 0 && bc.tunnel == 0) {
    // UDP faces can coexist if either port number differs
    return a->localUDP != b->localUDP || a->remoteUDP != b->remoteUDP;
  }
  if (a->localUDP != b->localUDP && a->remoteUDP != b->remoteUDP) {
    // UDP face and VXLAN/GTP-U face -or- two VXLAN/GTP-U faces can coexist if both port numbers
    // differ
    return true;
  }
  if (ac.tunnel != bc.tunnel) {
    // UDP face and VXLAN face and GTP-U face with same port numbers conflict
    return false;
  }
  if (ac.tunnel == 'V') {
    // VXLAN faces can coexist if VNI or inner MAC address differ
    return a->vxlan != b->vxlan || !rte_is_same_ether_addr(&a->innerLocal, &b->innerLocal) ||
           !rte_is_same_ether_addr(&a->innerRemote, &b->innerRemote);
  }
  if (ac.tunnel == 'G') {
    // GTP-U faces can coexist if TEID differ
    return a->ulTEID != b->ulTEID && a->dlTEID != b->dlTEID;
  }
  NDNDPDK_ASSERT(false);
}
