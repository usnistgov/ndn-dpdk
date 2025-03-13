#include "flowdef.h"
#include "../core/base16.h"
#include "../core/logger.h"
#include "hdr-impl.h"

N_LOG_INIT(EthFlowDef);

#define MASK(field) memset(&(field), 0xFF, sizeof(field))

__attribute__((nonnull)) static inline void
AppendItem(EthFlowDef* flow, size_t* i, enum rte_flow_item_type typ, const void* spec,
           const void* mask, size_t size) {
  flow->pattern[*i].type = typ;
  flow->pattern[*i].spec = spec;
  flow->pattern[*i].mask = mask;
  flow->patternSpecLen[*i] = size;
  ++(*i);
  NDNDPDK_ASSERT(*i < RTE_DIM(flow->pattern));
}

__attribute__((nonnull)) static inline void
PrepareRawItem(EthFlowDef* flow, int32_t offset, uint16_t length, const void* spec,
               const void* mask) {
  NDNDPDK_ASSERT(length <= sizeof(flow->rawSpecBuf));
  memmove(flow->rawSpecBuf, spec, length);
  memmove(flow->rawMaskBuf, mask, length);

  flow->rawSpec.relative = 1;
  flow->rawSpec.offset = offset;
  flow->rawSpec.length = length;
  flow->rawSpec.pattern = flow->rawSpecBuf;
  flow->rawMask = rte_flow_item_raw_mask;
  flow->rawMask.pattern = flow->rawMaskBuf;
}

__attribute__((nonnull)) static inline void
PrepareVxlan(const EthLocator* loc, struct rte_vxlan_hdr* vxlanSpec,
             struct rte_vxlan_hdr* vxlanMask, struct rte_ether_hdr* innerEthSpec,
             struct rte_ether_hdr* innerEthMask) {
  MASK(vxlanMask->vni);
  PutVxlanHdr((uint8_t*)vxlanSpec, loc->vxlan);

  MASK(innerEthMask->dst_addr);
  MASK(innerEthMask->src_addr);
  MASK(innerEthMask->ether_type);
  PutEtherHdr((uint8_t*)innerEthSpec, loc->innerRemote, loc->innerLocal, 0, EtherTypeNDN);
}

__attribute__((nonnull)) static inline EthFlowDefResult
GeneratePattern(EthFlowDef* flow, const EthLocator* loc, EthLocatorClass c, int variant) {
  size_t i = 0;
#define APPEND(typ, field)                                                                         \
  AppendItem(flow, &i, RTE_FLOW_ITEM_TYPE_##typ, &flow->field##Spec, &flow->field##Mask,           \
             sizeof(flow->field##Spec))

  if (c.passthru) {
    switch (variant) {
      case 0:
        flow->attr.priority = 1;
        return EthFlowDefResultValid;
      case 1:
        MASK(flow->ethMask.hdr.ether_type);
        flow->ethSpec.hdr.ether_type = rte_cpu_to_be_16(RTE_ETHER_TYPE_ARP);
        APPEND(ETH, eth);
        return EthFlowDefResultValid;
    }
    return 0;
  }

  MASK(flow->ethMask.hdr.dst_addr);
  PutEtherHdr((uint8_t*)(&flow->ethSpec.hdr), loc->remote, loc->local, loc->vlan, c.etherType);
  if (c.multicast) {
    flow->ethSpec.hdr.dst_addr = loc->remote;
  } else {
    MASK(flow->ethMask.hdr.src_addr);
  }
  APPEND(ETH, eth);

  if (loc->vlan != 0) {
    flow->vlanMask.hdr.vlan_tci = rte_cpu_to_be_16(0x0FFF); // don't mask PCP & DEI bits
    PutVlanHdr((uint8_t*)(&flow->vlanSpec.hdr), loc->vlan, c.etherType);
    APPEND(VLAN, vlan);
  }

  if (!c.udp) {
    // don't mask EtherType for IPv4/IPv6 - rejected by i40e driver
    MASK(flow->ethMask.hdr.ether_type);
    MASK(flow->vlanMask.hdr.eth_proto);
    return variant == 0 ? EthFlowDefResultValid : 0;
  }
  // i40e and several other drivers reject ETH+IP combination, so clear ETH spec
  flow->pattern[0].spec = NULL;

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
    MASK(flow->udpMask.hdr.src_port);
  }
  MASK(flow->udpMask.hdr.dst_port);
  PutUdpHdr((uint8_t*)(&flow->udpSpec.hdr), loc->remoteUDP, loc->localUDP);
  APPEND(UDP, udp);

  switch (c.tunnel) {
    case 'V': {
      switch (variant) {
        case 0:
          PrepareVxlan(loc, &flow->vxlanSpec.hdr, &flow->vxlanMask.hdr, &flow->innerEthSpec.hdr,
                       &flow->innerEthMask.hdr);
          APPEND(VXLAN, vxlan);
          APPEND(ETH, innerEth);
          return EthFlowDefResultValid;
        case 1: {
          struct {
            struct rte_vxlan_hdr vxlan;
            struct rte_ether_hdr eth;
          } __rte_aligned(2) spec = {0}, mask = {0};
          PrepareVxlan(loc, &spec.vxlan, &mask.vxlan, &spec.eth, &mask.eth);
          static_assert(sizeof(spec) == 4 + 16 + 2, "");
          PrepareRawItem(flow, 4, 16, RTE_PTR_ADD(&spec, 4), RTE_PTR_ADD(&mask, 4));
          APPEND(RAW, raw);
          return EthFlowDefResultValid;
        }
        default:
          return 0;
      }
    }
    case 'G': {
      EthGtpHdr spec = {0}, mask = {0};
      PutGtpHdr((uint8_t*)&spec, true, loc->ulTEID, loc->ulQFI);

      switch (variant) {
        case 0:
          APPEND(GTPU, gtp);
          goto FILL_GTP_ITEM;
        case 1:
          APPEND(GTP, gtp);
        FILL_GTP_ITEM:
          flow->gtpSpec.hdr = spec.hdr;
          MASK(flow->gtpMask.hdr.teid);
          return EthFlowDefResultValid;
        case 2:
          // In i40e driver, RAW item can have up to I40E_FDIR_MAX_FLEX_LEN=16 uint16 words, of
          // which up to I40E_FDIR_BITMASK_NUM_WORD=2 words may have a "bit mask" i.e. mask other
          // than 0000 and FFFF, see i40e_flow_store_flex_mask(). We use bit masks on the first and
          // eighth words. mask.hdr.ver is unmasked because masking it seems to cause packet loss.
          mask.hdr.pt = 1;
          mask.hdr.e = 1;
          MASK(mask.hdr.msg_type);
          MASK(mask.hdr.teid);
          mask.psc.qfi = 0b111111;
          static_assert(sizeof(spec) == 16);
          PrepareRawItem(flow, 0, 16, &spec, &mask);
          APPEND(RAW, raw);
          return EthFlowDefResultValid;
        default:
          return 0;
      }
    }
    default:
      return variant == 0 ? EthFlowDefResultValid : 0;
  }

#undef APPEND
}

__attribute__((nonnull)) static inline void
MaskSpecOctets(uint8_t* spec, const uint8_t* mask, size_t len) {
  for (size_t j = 0; j < len; ++j) {
    spec[j] &= mask[j];
  }
}

__attribute__((nonnull)) static inline void
CleanPattern(EthFlowDef* flow) {
  for (int i = 0;; ++i) {
    size_t itemLen = flow->patternSpecLen[i];
    struct rte_flow_item* item = &flow->pattern[i];
    switch (item->type) {
      case RTE_FLOW_ITEM_TYPE_END:
        return;
      case RTE_FLOW_ITEM_TYPE_RAW: {
        itemLen = offsetof(struct rte_flow_item_raw, pattern);
        const struct rte_flow_item_raw* spec = item->spec;
        const struct rte_flow_item_raw* mask = item->mask;
        MaskSpecOctets((uint8_t*)spec->pattern, mask->pattern, spec->length);
        break;
      }
      default:
        break;
    }

    if (item->spec == NULL) {
      item->mask = NULL;
      continue;
    }
    MaskSpecOctets((uint8_t*)item->spec, (const uint8_t*)item->mask, itemLen);
  }
}

__attribute__((nonnull)) static inline void
AppendAction(EthFlowDef* flow, size_t* i, enum rte_flow_action_type typ, const void* conf) {
  flow->actions[*i].type = typ;
  flow->actions[*i].conf = conf;
  ++(*i);
  NDNDPDK_ASSERT(*i < RTE_DIM(flow->pattern));
}

__attribute__((nonnull)) static inline EthFlowDefResult
GenerateActions(EthFlowDef* flow, EthLocatorClass c, int variant, uint32_t mark,
                const uint16_t queues[], int nQueues) {
  size_t i = 0;
#define APPEND(typ, field) AppendAction(flow, &i, RTE_FLOW_ACTION_TYPE_##typ, &flow->field##Act)

  NDNDPDK_ASSERT(nQueues >= 1);
  if (nQueues == 1) {
    flow->queueAct.index = queues[0];
    APPEND(QUEUE, queue);
  } else {
    flow->rssAct.level = 1;
    flow->rssAct.types = c.v4 ? RTE_ETH_RSS_NONFRAG_IPV4_UDP : RTE_ETH_RSS_NONFRAG_IPV6_UDP,
    flow->rssAct.queue_num = RTE_MIN((uint32_t)nQueues, RTE_DIM(flow->rssQueues));
    rte_memcpy(flow->rssQueues, queues, sizeof(queues[0]) * flow->rssAct.queue_num);
    flow->rssAct.queue = flow->rssQueues;
    APPEND(RSS, rss);
  }

  if (variant == 1) {
    return 0;
  }

  flow->markAct.id = mark;
  APPEND(MARK, mark);
  return EthFlowDefResultMarked;
#undef APPEND
}

EthFlowDefResult
EthFlowDef_Prepare(EthFlowDef* flow, const EthLocator* loc, int variant, uint32_t mark,
                   const uint16_t queues[], int nQueues) {
  NDNDPDK_ASSERT(variant < EthFlowDef_MaxVariant);
  EthLocatorClass c = EthLocator_Classify(loc);
  *flow = (const EthFlowDef){
    .attr.ingress = 1,
  };

  EthFlowDefResult res = GeneratePattern(flow, loc, c, variant / 2);
  if (!(res & EthFlowDefResultValid)) {
    return 0;
  }

  CleanPattern(flow);
  res |= GenerateActions(flow, c, variant % 2, mark, queues, nQueues);
  return res;
}

__attribute__((nonnull)) void
EthFlowDef_DebugPrint(const EthFlowDef* flow, const char* msg) {
  if (!N_LOG_ENABLED(DEBUG)) {
    return;
  }

  N_LOGD("%s", msg);
  N_LOGD("^ attr group=%" PRIu32 " priority=%" PRIu32, flow->attr.group, flow->attr.priority);

  for (int i = 0;; ++i) {
    const struct rte_flow_item* item = &flow->pattern[i];
    enum {
      b16BufOctets = 64,
      b16BufSize = Base16_BufferSize(b16BufOctets),
    };
    char b16Spec[b16BufSize] = {'-', 0};
    char b16Mask[b16BufSize] = {'-', 0};
    if (item->spec != NULL && item->mask != NULL) {
      NDNDPDK_ASSERT(flow->patternSpecLen[i] <= 64);
      Base16_Encode(b16Spec, sizeof(b16Spec), item->spec, flow->patternSpecLen[i]);
      Base16_Encode(b16Mask, sizeof(b16Mask), item->mask, flow->patternSpecLen[i]);
    }
    const char* typeName = NULL;
    if (rte_flow_conv(RTE_FLOW_CONV_OP_ITEM_NAME_PTR, &typeName, sizeof(&typeName),
                      (const void*)(uintptr_t)item->type, NULL) <= 0) {
      typeName = "-";
    }
    if (item->type == RTE_FLOW_ITEM_TYPE_RAW) {
      const struct rte_flow_item_raw* spec = item->spec;
      const struct rte_flow_item_raw* mask = item->mask;
      NDNDPDK_ASSERT(sizeof(*spec) + 1 + spec->length <= b16BufOctets);
      int b16Offset = 2 * sizeof(*spec);
      b16Spec[b16Offset] = '+';
      b16Mask[b16Offset] = '+';
      ++b16Offset;
      Base16_Encode(RTE_PTR_ADD(b16Spec, b16Offset), sizeof(b16Spec) - b16Offset, spec->pattern,
                    spec->length);
      Base16_Encode(RTE_PTR_ADD(b16Mask, b16Offset), sizeof(b16Mask) - b16Offset, mask->pattern,
                    spec->length);
    }
    N_LOGD("^ pattern index=%d type=%d~%s spec=%s mask=%s", i, (int)item->type, typeName, b16Spec,
           b16Mask);
    if (item->type == RTE_FLOW_ITEM_TYPE_END) {
      break;
    }
  }

  for (int i = 0;; ++i) {
    const struct rte_flow_action* action = &flow->actions[i];
    const char* typeName = NULL;
    if (rte_flow_conv(RTE_FLOW_CONV_OP_ACTION_NAME_PTR, &typeName, sizeof(&typeName),
                      (const void*)(uintptr_t)action->type, NULL) <= 0) {
      typeName = "-";
    }
    N_LOGD("^ action index=%d type=%d~%s", i, (int)action->type, typeName);
    if (action->type == RTE_FLOW_ACTION_TYPE_END) {
      break;
    }
  }
}

void
EthFlowDef_UpdateError(const EthFlowDef* flow, struct rte_flow_error* error) {
  ptrdiff_t offset = RTE_PTR_DIFF(error->cause, flow);
  if (offset >= 0 && (size_t)offset < sizeof(*flow)) {
    error->cause = (const void*)offset;
  }
}
