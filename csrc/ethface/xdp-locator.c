#include "xdp-locator.h"
#include "hdr-impl.h"

void
EthXdpLocator_Prepare(EthXdpLocator* xl, const EthLocator* loc) {
  EthLocatorClass c = EthLocator_Classify(loc);

  *xl = (const EthXdpLocator){0};
  if (c.etherType == 0) {
    return;
  }

  if (c.multicast) {
    rte_memcpy(xl->ether, &loc->remote, RTE_ETHER_ADDR_LEN);
  } else {
    rte_memcpy(xl->ether, &loc->local, RTE_ETHER_ADDR_LEN);
    rte_memcpy(RTE_PTR_ADD(xl->ether, RTE_ETHER_ADDR_LEN), &loc->remote, RTE_ETHER_ADDR_LEN);
  }
  if (loc->vlan != 0) {
    xl->vlan = rte_cpu_to_be_16(loc->vlan);
  }
  if (!c.udp) {
    return;
  }

  if (c.v4) {
    rte_memcpy(xl->ip, RTE_PTR_ADD(loc->remoteIP.a, V4_IN_V6_PREFIX_OCTETS),
               RTE_SIZEOF_FIELD(struct rte_ipv4_hdr, src_addr));
    rte_memcpy(RTE_PTR_ADD(xl->ip, RTE_SIZEOF_FIELD(struct rte_ipv4_hdr, src_addr)),
               RTE_PTR_ADD(loc->localIP.a, V4_IN_V6_PREFIX_OCTETS),
               RTE_SIZEOF_FIELD(struct rte_ipv4_hdr, dst_addr));
  } else {
    rte_memcpy(xl->ip, loc->remoteIP.a, RTE_IPV6_ADDR_SIZE);
    rte_memcpy(RTE_PTR_ADD(xl->ip, RTE_IPV6_ADDR_SIZE), loc->localIP.a, RTE_IPV6_ADDR_SIZE);
  }
  xl->udpSrc = rte_cpu_to_be_16(loc->remoteUDP);
  xl->udpDst = rte_cpu_to_be_16(loc->localUDP);
  switch (c.tunnel) {
    case 'V': {
      xl->udpSrc = 0;
      xl->vxlan = rte_cpu_to_be_32(loc->vxlan << 8);
      rte_memcpy(xl->inner, &loc->innerLocal, RTE_ETHER_ADDR_LEN);
      rte_memcpy(RTE_PTR_ADD(xl->inner, RTE_ETHER_ADDR_LEN), &loc->innerRemote, RTE_ETHER_ADDR_LEN);
      break;
    }
    case 'G': {
      xl->teid = rte_cpu_to_be_32(loc->ulTEID);
      xl->qfi = loc->ulQFI;
      break;
    }
  }
}
