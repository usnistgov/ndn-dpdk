#ifndef NDNDPDK_PDUMP_PARSE_H
#define NDNDPDK_PDUMP_PARSE_H

/** @file */

#include "../ndni/name.h"
#include "../ndni/tlv-decoder.h"

__attribute__((nonnull)) static inline LName
Pdump_ExtractNameL3_(TlvDecoder* d) {
  uint32_t length, type = TlvDecoder_ReadTL_MaybeTruncated(d, &length);
  if (unlikely(type != TtName)) {
    return (LName){0};
  }
  return (LName){
    .value = rte_pktmbuf_mtod_offset(d->m, const uint8_t*, d->offset),
    .length = RTE_MIN(length, d->m->data_len - d->offset),
  };
}

/**
 * @brief Extract Interest/Data name from mbuf.
 *
 * If @p pkt is an Interest/Data packet, extract its name.
 * If @p pkt is the first fragment of an Interest/Data packet, extract the portion of name
 * contained in this fragment; it may be truncated and contain incomplete name component.
 * Otherwise, return empty LName.
 */
__attribute__((nonnull)) static inline LName
Pdump_ExtractName(struct rte_mbuf* pkt) {
  TlvDecoder d = TlvDecoder_Init(pkt);
  uint32_t length0, type0 = TlvDecoder_ReadTL(&d, &length0);
  switch (type0) {
    case TtInterest:
    case TtData:
      return Pdump_ExtractNameL3_(&d);
    case TtLpPacket:
      break;
    default:
      goto NO_MATCH;
  }

  TlvDecoder_EachTL (&d, type1, length1) {
    switch (type1) {
      case TtFragIndex: {
        uint8_t fragIndex = 0;
        if (unlikely(!TlvDecoder_ReadNniTo(&d, length1, &fragIndex)) || fragIndex > 0) {
          goto NO_MATCH;
        }
        break;
      }
      case TtLpPayload: {
        uint32_t length2, type2 = TlvDecoder_ReadTL_MaybeTruncated(&d, &length2);
        switch (type2) {
          case TtInterest:
          case TtData: {
            return Pdump_ExtractNameL3_(&d);
          }
          default:
            goto NO_MATCH;
        }
      }
      default:
        TlvDecoder_Skip(&d, length1);
        break;
    }
  }

NO_MATCH:;
  return (LName){0};
}

#endif // NDNDPDK_PDUMP_PARSE_H
