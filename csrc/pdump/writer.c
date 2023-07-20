#include "writer.h"
#include "../iface/faceid.h"
#include "format.h"

__attribute__((nonnull)) static __rte_noinline void
WriteBlock(PdumpWriter* w, struct rte_mbuf* pkt) {
  NDNDPDK_ASSERT(pkt->pkt_len % 4 == 0);
  NDNDPDK_ASSERT(pkt->pkt_len == pkt->data_len);
  if (unlikely(w->pos + pkt->pkt_len > w->m.size)) {
    w->full = true;
    return;
  }
  rte_memcpy(MmapFd_At(&w->m, w->pos), rte_pktmbuf_mtod(pkt, const uint8_t*), pkt->pkt_len);
  w->pos += pkt->pkt_len;
}

/**
 * @brief Write PCAPNG enhanced packet block.
 * @param epb buffer for EPB header.
 * @param hdrLen sizeof EPB header and packet header before mbuf payload.
 */
__attribute__((nonnull)) static __rte_always_inline void
WriteEPB(PdumpWriter* w, struct rte_mbuf* pkt, PcapngEPB* epb, size_t hdrLen) {
  uint32_t intf = w->intf[pkt->port];
  if (unlikely(intf == UINT32_MAX)) {
    return;
  }

  uint32_t totalLength = hdrLen + pkt->pkt_len + sizeof(PcapngTrailer);
  totalLength = (totalLength + 0x03) & (~0x03);
  if (unlikely(w->pos + totalLength > w->m.size)) {
    w->full = true;
    return;
  }

  uint64_t time = TscTime_ToUnixNano(Mbuf_GetTimestamp(pkt));
  uint32_t pktLen = hdrLen - sizeof(PcapngEPB) + pkt->pkt_len;
  *epb = (PcapngEPB){
    .blockType = rte_cpu_to_le_32(PdumpNgTypeEPB),
    .totalLength = rte_cpu_to_le_32(totalLength),
    .intf = rte_cpu_to_le_32(intf),
    .timeHi = rte_cpu_to_le_32(time >> 32),
    .timeLo = rte_cpu_to_le_32(time & UINT32_MAX),
    .capLen = rte_cpu_to_le_32(pktLen),
    .origLen = rte_cpu_to_le_32(pktLen),
  };
  rte_memcpy(MmapFd_At(&w->m, w->pos), epb, hdrLen);

  uint8_t* dst = MmapFd_At(&w->m, w->pos + hdrLen);
  Mbuf_ReadTo(pkt, 0, pkt->pkt_len, dst);

  PcapngTrailer trailer = {
    .totalLength = epb->totalLength,
  };
  rte_memcpy(MmapFd_At(&w->m, w->pos + totalLength - sizeof(trailer)), &trailer, sizeof(trailer));

  w->pos += totalLength;
}

__attribute__((nonnull)) static inline void
WriteRaw(PdumpWriter* w, struct rte_mbuf* pkt) {
  PcapngEPB epb = {0};
  WriteEPB(w, pkt, &epb, sizeof(epb));
}

__attribute__((nonnull)) static inline void
WriteSLL(PdumpWriter* w, struct rte_mbuf* pkt) {
  PcapngEPBSLL hdr = {
    .sll =
      {
        .sll_pkttype = pkt->packet_type, // take lower 16bits
        .sll_hatype = rte_cpu_to_be_16(UINT16_MAX),
        .sll_protocol = rte_cpu_to_be_16(EtherTypeNDN),
      },
  };
  WriteEPB(w, pkt, &hdr.epb, sizeof(hdr));
}

__attribute__((nonnull)) static inline void
ProcessMbuf(PdumpWriter* w, struct rte_mbuf* pkt) {
  switch (pkt->packet_type) {
    case PdumpMbufTypeRaw:
      WriteRaw(w, pkt);
      break;
    case PdumpMbufTypeSLL | SLLIncoming:
    case PdumpMbufTypeSLL | SLLOutgoing:
      WriteSLL(w, pkt);
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
}

int
PdumpWriter_Run(PdumpWriter* w) {
  if (!MmapFd_Open(&w->m, w->filename, w->maxSize)) {
    return 1;
  }

  uint16_t count = 0;
  while (ThreadCtrl_Continue(w->ctrl, count) && !w->full) {
    struct rte_mbuf* pkts[PdumpWriterBurstSize];
    count = rte_ring_dequeue_burst(w->queue, (void**)pkts, RTE_DIM(pkts), NULL);

    for (uint16_t i = 0; i < count && !w->full; ++i) {
      ProcessMbuf(w, pkts[i]);
    }
    rte_pktmbuf_free_bulk(pkts, count);
  }

  if (!MmapFd_Close(&w->m, w->filename, w->pos)) {
    return 2;
  }
  return 0;
}
