#include "flow-pattern.h"
#include "hdr-impl.h"

static void
EthFlowPattern_Set(EthFlowPattern* flow, size_t i, enum rte_flow_item_type typ, uint8_t* spec,
                   uint8_t* mask, size_t size) {
  for (size_t j = 0; j < size; ++j) {
    spec[j] &= mask[j];
  }
  flow->pattern[i].type = typ;
  flow->pattern[i].spec = spec;
  flow->pattern[i].mask = mask;
}

void
EthFlowPattern_Prepare(EthFlowPattern* flow, uint32_t* priority, const EthLocator* loc,
                       bool prefersFlowItemGTP) {
  EthLocatorClass c = EthLocator_Classify(loc);

  *flow = (const EthFlowPattern){0};
  flow->pattern[0].type = RTE_FLOW_ITEM_TYPE_END;
  *priority = 0;
  size_t i = 0;
#define MASK(field) memset(&(field), 0xFF, sizeof(field))
#define APPEND(typ, field)                                                                         \
  do {                                                                                             \
    EthFlowPattern_Set(flow, i, RTE_FLOW_ITEM_TYPE_##typ, (uint8_t*)&flow->field##Spec,            \
                       (uint8_t*)&flow->field##Mask, sizeof(flow->field##Mask));                   \
    ++i;                                                                                           \
    NDNDPDK_ASSERT(i < RTE_DIM(flow->pattern));                                                    \
    flow->pattern[i].type = RTE_FLOW_ITEM_TYPE_END;                                                \
  } while (false)

  if (c.passthru) {
    *priority = 1;
    return;
  }

  MASK(flow->ethMask.hdr.dst_addr);
  MASK(flow->ethMask.hdr.ether_type);
  PutEtherHdr((uint8_t*)(&flow->ethSpec.hdr), loc->remote, loc->local, loc->vlan, c.etherType);
  if (c.multicast) {
    flow->ethSpec.hdr.dst_addr = loc->remote;
  } else {
    MASK(flow->ethMask.hdr.src_addr);
  }
  APPEND(ETH, eth);

  if (loc->vlan != 0) {
    flow->vlanMask.hdr.vlan_tci = rte_cpu_to_be_16(0x0FFF); // don't mask PCP & DEI bits
    MASK(flow->vlanMask.hdr.eth_proto);
    PutVlanHdr((uint8_t*)(&flow->vlanSpec.hdr), loc->vlan, c.etherType);
    APPEND(VLAN, vlan);
  }

  if (!c.udp) {
    MASK(flow->vlanMask.hdr.eth_proto);
    return;
  }
  // several drivers do not support ETH+IP combination, so clear ETH spec
  flow->pattern[0].spec = NULL;
  flow->pattern[0].mask = NULL;

  if (c.v4) {
    MASK(flow->ip4Mask.hdr.src_addr);
    MASK(flow->ip4Mask.hdr.dst_addr);
    PutIpv4Hdr((uint8_t*)(&flow->ip4Spec.hdr), loc->remoteIP, loc->localIP);
    APPEND(IPV4, ip4);
  } else {
    MASK(flow->ip6Mask.hdr.src_addr);
    MASK(flow->ip6Mask.hdr.dst_addr);
    PutIpv6Hdr((uint8_t*)(&flow->ip6Spec.hdr), loc->remoteIP, loc->localIP);
    APPEND(IPV6, ip6);
  }

  if (c.tunnel != 'V') { // VXLAN packet can have any UDP source port
    MASK(flow->udpMask.hdr.dst_port);
  }
  MASK(flow->udpMask.hdr.src_port);
  PutUdpHdr((uint8_t*)(&flow->udpSpec.hdr), loc->remoteUDP, loc->localUDP);
  APPEND(UDP, udp);

  switch (c.tunnel) {
    case 'V': {
      MASK(flow->vxlanMask.hdr.vni);
      PutVxlanHdr((uint8_t*)(&flow->vxlanSpec.hdr), loc->vxlan);
      APPEND(VXLAN, vxlan);

      MASK(flow->innerEthMask.hdr.dst_addr);
      MASK(flow->innerEthMask.hdr.src_addr);
      MASK(flow->innerEthMask.hdr.ether_type);
      PutEtherHdr((uint8_t*)(&flow->innerEthSpec.hdr), loc->innerRemote, loc->innerLocal, 0,
                  EtherTypeNDN);
      APPEND(ETH, innerEth);
      break;
    }
    case 'G': {
      MASK(flow->gtpMask.hdr.teid);
      PutGtpHdrMinimal(&flow->gtpSpec.hdr, loc->ulTEID);
      if (prefersFlowItemGTP) {
        APPEND(GTP, gtp);
      } else {
        APPEND(GTPU, gtp);
      }
      break;
    }
  }

#undef MASK
#undef APPEND
}
