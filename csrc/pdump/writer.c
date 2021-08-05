#include "writer.h"
#include "../iface/faceid.h"
#include "format.h"

__attribute__((nonnull)) static __rte_noinline void
WriteBlock(PdumpWriter* w, struct rte_mbuf* pkt)
{
  NDNDPDK_ASSERT(pkt->pkt_len % 4 == 0);
  NDNDPDK_ASSERT(pkt->pkt_len == pkt->data_len);
  rte_memcpy(MmapFd_At(&w->m, w->pos), rte_pktmbuf_mtod(pkt, const uint8_t*), pkt->data_len);
  w->pos += pkt->data_len;
}

__attribute__((nonnull)) static inline void
WriteSLL(PdumpWriter* w, struct rte_mbuf* pkt, uint32_t len4)
{
  uint32_t intf = w->intf[pkt->port];
  if (unlikely(intf == UINT32_MAX)) {
    return;
  }

  uint64_t time = TscTime_ToUnixNano(Mbuf_GetTimestamp(pkt));
  rte_le32_t pktLen = SLL_HDR_LEN + pkt->pkt_len;
  PcapngEPBSLL hdr = {
    .epb = {
      .blockType = rte_cpu_to_le_32(PdumpNgTypeEPB),
      .totalLength = rte_cpu_to_le_32(sizeof(PcapngEPBSLL) + len4 + sizeof(rte_le32_t)),
      .intf = rte_cpu_to_le_32(intf),
      .timeHi = rte_cpu_to_le_32(time >> 32),
      .timeLo = rte_cpu_to_le_32(time & UINT32_MAX),
      .capLen = rte_cpu_to_le_32(pktLen),
      .origLen = rte_cpu_to_le_32(pktLen),
    },
    .sll = {
      .sll_pkttype = pkt->packet_type,
      .sll_hatype = rte_cpu_to_be_16(UINT16_MAX),
      .sll_protocol = rte_cpu_to_be_16(EtherTypeNDN),
    },
  };
  rte_memcpy(MmapFd_At(&w->m, w->pos), &hdr, sizeof(hdr));
  uint8_t* dst = MmapFd_At(&w->m, w->pos + sizeof(hdr));
  if (likely(pkt->pkt_len <= pkt->data_len)) {
    rte_memcpy(dst, rte_pktmbuf_mtod(pkt, const uint8_t*), pkt->data_len);
  } else {
    const uint8_t* readTo = rte_pktmbuf_read(pkt, 0, pkt->pkt_len, dst);
    NDNDPDK_ASSERT(readTo == dst);
  }
  *(rte_le32_t*)RTE_PTR_ADD(dst, len4) = hdr.epb.totalLength;
  w->pos += sizeof(PcapngEPBSLL) + len4 + sizeof(rte_le32_t);
}

__attribute__((nonnull)) static inline bool
ProcessMbuf(PdumpWriter* w, struct rte_mbuf* pkt)
{
  uint32_t len4 = (pkt->pkt_len + 0x03) & (~0x03);
  if (w->pos + sizeof(PcapngEPBSLL) + len4 + sizeof(rte_le32_t) > w->m.size) {
    return true;
  }

  switch (pkt->packet_type) {
    case SLLIncoming:
    case SLLOutgoing:
      WriteSLL(w, pkt, len4);
      break;
    case PdumpNgTypeIDB:
      w->intf[pkt->port] = w->nextIntf++;
      // fallthrough
    case PdumpNgTypeSHB:
      WriteBlock(w, pkt);
      break;
    default:
      NDNDPDK_ASSERT(false);
      break;
  }
  return false;
}

int
PdumpWriter_Run(PdumpWriter* w)
{
  if (!MmapFd_Open(&w->m, w->filename, w->maxSize)) {
    return 1;
  }

  while (ThreadStopFlag_ShouldContinue(&w->stop)) {
    struct rte_mbuf* pkts[PdumpWriterBurstSize];
    uint16_t count = rte_ring_dequeue_burst(w->queue, (void**)pkts, RTE_DIM(pkts), NULL);

    bool full = false;
    for (uint16_t i = 0; i < count; ++i) {
      full = ProcessMbuf(w, pkts[i]);
      if (unlikely(full)) {
        break;
      }
    }

    rte_pktmbuf_free_bulk(pkts, count);
    if (unlikely(full)) {
      break;
    }
  }

  if (!MmapFd_Close(&w->m, w->pos)) {
    return 2;
  }
  return 0;
}
