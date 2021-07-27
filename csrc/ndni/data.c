#include "data.h"
#include "packet.h"
#include "tlv-decoder.h"
#include "tlv-encoder.h"

static __rte_always_inline bool
PData_ParseMetaInfo_(PData* data, TlvDecoder* d)
{
  TlvDecoder_EachTL (d, type, length) {
    switch (type) {
      case TtContentType:
      case TtFinalBlock:
        TlvDecoder_Skip(d, length);
        break;
      case TtFreshnessPeriod: {
        if (unlikely(!TlvDecoder_ReadNniTo(d, length, &data->freshness))) {
          return false;
        }
        return true;
      }
      default:
        if (TlvDecoder_IsCriticalType(type)) {
          return false;
        }
        TlvDecoder_Skip(d, length);
        break;
    }
  }
  return true;
}

bool
PData_Parse(PData* data, struct rte_mbuf* pkt)
{
  NDNDPDK_ASSERT(RTE_MBUF_DIRECT(pkt) && rte_mbuf_refcnt_read(pkt) == 1);
  *data = (const PData){ 0 };

  TlvDecoder d;
  TlvDecoder_Init(&d, pkt);
  uint32_t length0, type0 = TlvDecoder_ReadTL(&d, &length0);
  NDNDPDK_ASSERT(type0 == TtData);

  TlvDecoder_EachTL (&d, type, length) {
    switch (type) {
      case TtName: {
        LName lname = (LName){ .length = length };
        if (unlikely(length > NameMaxLength ||
                     (lname.value = TlvDecoder_Linearize(&d, length)) == NULL)) {
          return false;
        }
        if (unlikely(!PName_Parse(&data->name, lname))) {
          return false;
        }
        break;
      }
      case TtMetaInfo: {
        TlvDecoder vd;
        TlvDecoder_MakeValueDecoder(&d, length, &vd);
        if (unlikely(!PData_ParseMetaInfo_(data, &vd))) {
          return false;
        }
        break;
      }
      case TtContent:
      case TtDSigInfo:
      case TtDSigValue: {
        return true;
      }
      default:
        if (TlvDecoder_IsCriticalType(type)) {
          return false;
        }
        TlvDecoder_Skip(&d, length);
        break;
    }
  }

  return true;
}

__attribute__((nonnull)) static DataSatisfyResult
PData_CanSatisfy_HasDigestComp_(PData* data, PInterest* interest)
{
  if (interest->name.length != data->name.length + ImplicitDigestSize ||
      memcmp(interest->name.value, data->name.value, data->name.length) != 0) {
    return DataSatisfyNo;
  }

  if (!data->hasDigest) {
    return DataSatisfyNeedDigest;
  }

  return memcmp(RTE_PTR_ADD(interest->name.value, interest->name.length - ImplicitDigestLength),
                data->digest, ImplicitDigestLength) == 0
           ? DataSatisfyYes
           : DataSatisfyNo;
}

DataSatisfyResult
PData_CanSatisfy(PData* data, PInterest* interest)
{
  if (unlikely(interest->mustBeFresh && data->freshness == 0)) {
    return DataSatisfyNo;
  }

  if (unlikely(interest->name.hasDigestComp)) {
    return PData_CanSatisfy_HasDigestComp_(data, interest);
  }

  int cmp = LName_IsPrefix(PName_ToLName(&interest->name), PName_ToLName(&data->name));
  if (interest->canBePrefix) {
    return cmp >= 0 ? DataSatisfyYes : DataSatisfyNo;
  }
  return cmp == 0 ? DataSatisfyYes : DataSatisfyNo;
}

void
DataDigest_Prepare(Packet* npkt, struct rte_crypto_op* op)
{
  PData* data = Packet_GetDataHdr(npkt);
  struct rte_mbuf* pkt = Packet_ToMbuf(npkt);
  CryptoOp_PrepareSha256Digest(op, pkt, 0, pkt->pkt_len, data->digest);
}

uint16_t
DataDigest_Enqueue(CryptoQueuePair cqp, struct rte_crypto_op** ops, uint16_t count)
{
  if (unlikely(count == 0)) {
    return 0;
  }

  uint16_t nEnq = rte_cryptodev_enqueue_burst(cqp.dev, cqp.qp, ops, count);
  for (uint16_t i = nEnq; i < count; ++i) {
    Packet* npkt = DataDigest_Finish(ops[i]);
    NDNDPDK_ASSERT(npkt == NULL);
  }
  return count - nEnq;
}

Packet*
DataDigest_Finish(struct rte_crypto_op* op)
{
  if (unlikely(op->status != RTE_CRYPTO_OP_STATUS_SUCCESS)) {
    rte_pktmbuf_free(op->sym->m_src);
    rte_crypto_op_free(op);
    return NULL;
  }

  Packet* npkt = Packet_FromMbuf(op->sym->m_src);
  PData* data = Packet_GetDataHdr(npkt);
  data->hasDigest = true;
  rte_crypto_op_free(op);
  return npkt;
}

__attribute__((nonnull, returns_nonnull)) static inline Packet*
Encode_Finish(struct rte_mbuf* m)
{
  TlvEncoder_PrependTL(m, TtData, m->pkt_len);

  Packet* output = Packet_FromMbuf(m);
  Packet_SetType(output, PktSData);
  *Packet_GetLpL3Hdr(output) = (const LpL3){ 0 };
  return output;
}

__attribute__((nonnull)) static Packet*
Encode_Linear(DataGen* gen, LName prefix, PacketMempools* mp, uint16_t fragmentPayloadSize)
{
  uint32_t pktLen = L3TypeLengthHeadroom + L3TypeLengthHeadroom + // Data TL + Name TL
                    prefix.length + gen->tpl->pkt_len;
  uint32_t fragCount = DIV_CEIL(pktLen, fragmentPayloadSize);
  NDNDPDK_ASSERT(fragCount < LpMaxFragments);
  struct rte_mbuf* frames[LpMaxFragments];
  if (unlikely(rte_pktmbuf_alloc_bulk(mp->packet, frames, fragCount) != 0)) {
    return NULL;
  }

  uint32_t fragIndex = 0;
  uint16_t extraHeadroom = L3TypeLengthHeadroom + L3TypeLengthHeadroom; // Data TL + Name TL
  for (uint16_t offset = 0; offset < prefix.length;) {
    frames[fragIndex]->data_off = RTE_PKTMBUF_HEADROOM + LpHeaderHeadroom + extraHeadroom;
    uint16_t fragSize = RTE_MIN(prefix.length - offset, fragmentPayloadSize - extraHeadroom);
    rte_memcpy(rte_pktmbuf_append(frames[fragIndex], fragSize), RTE_PTR_ADD(prefix.value, offset),
               fragSize);
    extraHeadroom = 0;
    offset += fragSize;
  }
  TlvEncoder_PrependTL(frames[0], TtName, prefix.length + gen->suffixL);

  TlvDecoder d;
  TlvDecoder_Init(&d, gen->tpl);
  TlvDecoder_Fragment(&d, d.length, frames, &fragIndex, fragCount, fragmentPayloadSize,
                      RTE_PKTMBUF_HEADROOM + LpHeaderHeadroom);

  Mbuf_ChainVector(frames, fragCount);
  return Encode_Finish(frames[0]);
}

__attribute__((nonnull)) static Packet*
Encode_Chained(DataGen* gen, LName prefix, PacketMempools* mp)
{
  struct rte_mbuf* seg1 = rte_pktmbuf_alloc(mp->indirect);
  if (unlikely(seg1 == NULL)) {
    return NULL;
  }
  rte_pktmbuf_attach(seg1, gen->tpl);

  struct rte_mbuf* seg0 = rte_pktmbuf_alloc(mp->header);
  if (unlikely(seg0 == NULL)) {
    rte_pktmbuf_free(seg1);
    return NULL;
  }
  seg0->data_off = RTE_PKTMBUF_HEADROOM + LpHeaderHeadroom +    //
                   L3TypeLengthHeadroom + L3TypeLengthHeadroom; // Data TL + Name TL
  rte_memcpy(rte_pktmbuf_append(seg0, prefix.length), prefix.value, prefix.length);
  TlvEncoder_PrependTL(seg0, TtName, prefix.length + gen->suffixL);

  bool ok = Mbuf_Chain(seg0, seg0, seg1);
  NDNDPDK_ASSERT(ok);
  return Encode_Finish(seg0);
}

__attribute__((nonnull)) Packet*
DataGen_Encode(DataGen* gen, LName prefix, PacketMempools* mp, PacketTxAlign align)
{
  if (align.linearize) {
    return Encode_Linear(gen, prefix, mp, align.fragmentPayloadSize);
  }
  return Encode_Chained(gen, prefix, mp);
}
