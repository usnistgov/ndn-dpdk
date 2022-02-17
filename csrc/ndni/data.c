#include "data.h"
#include "packet.h"
#include "tlv-decoder.h"
#include "tlv-encoder.h"

static struct
{
  unaligned_uint16_t sigInfoTL;
  unaligned_uint16_t sigTypeTL;
  uint8_t sigTypeV;
  unaligned_uint16_t sigValueTL;
} __rte_packed NullSig;

RTE_INIT(InitNullSig)
{
  NullSig.sigInfoTL =
    TlvEncoder_ConstTL1(TtDSigInfo, sizeof(NullSig.sigTypeTL) + sizeof(NullSig.sigTypeV));
  NullSig.sigTypeTL = TlvEncoder_ConstTL1(TtSigType, sizeof(NullSig.sigTypeV));
  NullSig.sigTypeV = SigNull;
  NullSig.sigValueTL = TlvEncoder_ConstTL1(TtDSigValue, 0);

  static_assert(sizeof(NullSig) == DataEncNullSigLen, "");
}

__attribute__((nonnull)) static __rte_always_inline bool
PData_ParseMetaInfo(PData* data, TlvDecoder* d)
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
        if (unlikely(!PData_ParseMetaInfo(data, &vd))) {
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

bool
DataEnc_PrepareMetaInfo_(void* metaBuf, size_t capacity, ContentType ct, uint32_t freshness,
                         LName finalBlock)
{
  DataEnc_MetaInfoBuffer(0)* meta = metaBuf;
  meta->size = 2;
#define APPEND(ptr)                                                                                \
  do {                                                                                             \
    if (unlikely((size_t)meta->size + sizeof(*ptr) > capacity)) {                                  \
      return false;                                                                                \
    }                                                                                              \
    ptr = RTE_PTR_ADD(meta->value, meta->size);                                                    \
    meta->size += sizeof(*ptr);                                                                    \
  } while (false)

  if (unlikely(ct != ContentBlob)) {
    struct ContentTypeF
    {
      unaligned_uint16_t contentTypeTL;
      uint8_t contentTypeV;
    } __rte_packed* f = NULL;
    APPEND(f);
    f->contentTypeTL = TlvEncoder_ConstTL1(TtContentType, sizeof(f->contentTypeV));
    f->contentTypeV = ct;
  }
  if (freshness > 0) {
    struct FreshnessF
    {
      unaligned_uint16_t freshnessTL;
      unaligned_uint32_t freshnessV;
    } __rte_packed* f = NULL;
    APPEND(f);
    f->freshnessTL = TlvEncoder_ConstTL1(TtFreshnessPeriod, sizeof(f->freshnessV));
    f->freshnessV = rte_cpu_to_be_32(freshness);
  }
  if (finalBlock.length > 0) {
    struct FinalBlockF
    {
      uint8_t finalBlockT;
      uint8_t finalBlockL;
      uint8_t finalBlockV[];
    } __rte_packed* f = NULL;
    APPEND(f);
    if (unlikely((size_t)meta->size + finalBlock.length > capacity)) {
      return false;
    }
    meta->size += finalBlock.length;
    f->finalBlockT = TtFinalBlock;
    f->finalBlockL = finalBlock.length;
    rte_memcpy(f->finalBlockV, finalBlock.value, finalBlock.length);
  }
  meta->value[0] = TtMetaInfo;
  meta->value[1] = meta->size - 2;
#undef APPEND
  return true;
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

Packet*
DataEnc_EncodePayload(LName prefix, LName suffix, const void* metaBuf, struct rte_mbuf* m)
{
  NDNDPDK_ASSERT(RTE_MBUF_DIRECT(m) && rte_pktmbuf_is_contiguous(m) &&
                 rte_mbuf_refcnt_read(m) == 1);
  const DataEnc_MetaInfoBuffer(0)* meta = metaBuf;

  uint16_t nameL = prefix.length + suffix.length;
  uint16_t sizeofNameL = TlvEncoder_SizeofVarNum(nameL);
  uint32_t contentL = m->pkt_len;
  uint16_t sizeofContentL = TlvEncoder_SizeofVarNum(contentL);
  uint16_t sizeofHeadroom = 1 + sizeofNameL + nameL + meta->size + 1 + sizeofContentL;

  uint8_t* sig = (uint8_t*)rte_pktmbuf_append(m, sizeof(NullSig));
  if (unlikely(sig == NULL || rte_pktmbuf_headroom(m) < 4 + sizeofHeadroom)) {
    return NULL;
  }
  rte_memcpy(sig, &NullSig, sizeof(NullSig));

  uint8_t* head = (uint8_t*)rte_pktmbuf_prepend(m, sizeofHeadroom);
  *head++ = TtName;
  head += TlvEncoder_WriteVarNum(head, nameL);
  rte_memcpy(head, prefix.value, prefix.length);
  head += prefix.length;
  rte_memcpy(head, suffix.value, suffix.length);
  head += suffix.length;
  rte_memcpy(head, meta->value, meta->size);
  head += meta->size;
  *head++ = TtContent;
  head += TlvEncoder_WriteVarNum(head, contentL);

  return Encode_Finish(m);
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
