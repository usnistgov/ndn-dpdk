#include "data.h"
#include "packet.h"
#include "tlv-decoder.h"
#include "tlv-encoder.h"

// helperScratch should be small enough not to increase PacketPriv size
static_assert(sizeof(PData) <= sizeof(PInterest), "");

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

__attribute__((nonnull)) static inline bool
PData_ParseMetaInfo(PData* data, TlvDecoder* d, ParseFor parseFor)
{
  TlvDecoder_EachTL (d, type, length) {
    switch (type) {
      case TtFreshnessPeriod: {
        if (unlikely(!TlvDecoder_ReadNniTo(d, length, &data->freshness))) {
          return false;
        }
        break;
      }
      case TtFinalBlock: {
        if (parseFor == ParseForFw) {
          TlvDecoder_Skip(d, length);
        } else {
          LName lastComp = PName_Slice(&data->name, -1, INT16_MAX);
          if (likely(lastComp.length == length)) {
            uint8_t scratch[NameMaxLength];
            const uint8_t* finalBlockComp = TlvDecoder_Read(d, scratch, lastComp.length);
            data->isFinalBlock = memcmp(lastComp.value, finalBlockComp, lastComp.length) == 0;
          } else {
            TlvDecoder_Skip(d, length);
          }
        }
        break;
      }
      default:
        if (TlvDecoder_IsCriticalType(type)) {
          return false;
        }
        // fallthrough
      case TtContentType:
        TlvDecoder_Skip(d, length);
        break;
    }
  }
  return true;
}

bool
PData_Parse(PData* data, struct rte_mbuf* pkt, ParseFor parseFor)
{
  NDNDPDK_ASSERT(RTE_MBUF_DIRECT(pkt) && rte_mbuf_refcnt_read(pkt) == 1);
  *data = (const PData){ 0 };

  TlvDecoder d = TlvDecoder_Init(pkt);
  uint32_t length0, type0 = TlvDecoder_ReadTL(&d, &length0);
  NDNDPDK_ASSERT(type0 == TtData);

  TlvDecoder_EachTL (&d, type, length) {
    switch (type) {
      case TtName: {
        LName lname = (LName){ .length = length };
        if (unlikely(length > NameMaxLength ||
                     (lname.value = TlvDecoder_Linearize(&d, length)) == NULL ||
                     !PName_Parse(&data->name, lname))) {
          return false;
        }
        break;
      }
      case TtMetaInfo: {
        TlvDecoder vd = TlvDecoder_MakeValueDecoder(&d, length);
        if (unlikely(!PData_ParseMetaInfo(data, &vd, parseFor))) {
          return false;
        }
        break;
      }
      case TtContent: {
        data->contentOffset = pkt->pkt_len - d.length;
        data->contentL = length;
      }
      // fallthrough
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

struct rte_crypto_op*
DataDigest_Prepare(CryptoQueuePair* cqp, Packet* npkt)
{
  PData* data = Packet_GetDataHdr(npkt);
  static_assert(sizeof(struct rte_crypto_op) + sizeof(struct rte_crypto_sym_op) <=
                  sizeof(data->helperScratch),
                "");
  struct rte_crypto_op* op = (void*)data->helperScratch;
  op->mempool = NULL;
  op->phys_addr = 0;

  struct rte_mbuf* m = Packet_ToMbuf(npkt);
  CryptoQueuePair_PrepareSha256(cqp, op, m, 0, m->pkt_len, data->digest);
  return op;
}

uint16_t
DataDigest_Enqueue(CryptoQueuePair* cqp, struct rte_crypto_op** ops, uint16_t count)
{
  if (unlikely(count == 0)) {
    return 0;
  }
  uint16_t nEnq = rte_cryptodev_enqueue_burst(cqp->dev, cqp->qp, ops, count);
  return count - nEnq;
}

bool
DataDigest_Finish(struct rte_crypto_op* op, Packet** npkt)
{
  NDNDPDK_ASSERT(op->mempool == NULL);
  *npkt = Packet_FromMbuf(op->sym->m_src);
  PData* data = Packet_GetDataHdr(*npkt);
  data->hasDigest = op->status == RTE_CRYPTO_OP_STATUS_SUCCESS;
  return data->hasDigest;
}

void
DataEnc_PrepareMetaInfo(uint8_t* room, ContentType ct, uint32_t freshness, LName finalBlock)
{
  room[0] = TtMetaInfo;
  room[1] = 0;
#define APPEND(ptr, extraLength)                                                                   \
  do {                                                                                             \
    ptr = RTE_PTR_ADD(room, 2 + room[1]);                                                          \
    room[1] += sizeof(*ptr) + (extraLength);                                                       \
  } while (false)

  if (unlikely(ct != ContentBlob)) {
    struct ContentTypeF
    {
      unaligned_uint16_t contentTypeTL;
      uint8_t contentTypeV;
    } __rte_packed* f = NULL;
    APPEND(f, 0);
    f->contentTypeTL = TlvEncoder_ConstTL1(TtContentType, sizeof(f->contentTypeV));
    f->contentTypeV = ct;
  }

  if (freshness > 0) {
    struct FreshnessF
    {
      unaligned_uint16_t freshnessTL;
      unaligned_uint32_t freshnessV;
    } __rte_packed* f = NULL;
    APPEND(f, 0);
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
    APPEND(f, finalBlock.length);
    f->finalBlockT = TtFinalBlock;
    f->finalBlockL = finalBlock.length;
    rte_memcpy(f->finalBlockV, finalBlock.value, finalBlock.length);
  }

#undef APPEND
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
DataEnc_EncodePayload(LName prefix, LName suffix, const uint8_t* meta, struct rte_mbuf* m)
{
  NDNDPDK_ASSERT(RTE_MBUF_DIRECT(m) && rte_pktmbuf_is_contiguous(m) &&
                 rte_mbuf_refcnt_read(m) == 1);

  uint16_t nameL = prefix.length + suffix.length;
  uint16_t sizeofNameL = TlvEncoder_SizeofVarNum(nameL);
  uint16_t sizeofMeta = DataEnc_SizeofMetaInfo(meta);
  uint32_t contentL = m->pkt_len;
  uint16_t sizeofContentL = TlvEncoder_SizeofVarNum(contentL);
  uint16_t sizeofHeadroom = 1 + sizeofNameL + nameL + sizeofMeta + 1 + sizeofContentL;

  uint8_t* sig = (uint8_t*)rte_pktmbuf_append(m, sizeof(NullSig));
  if (unlikely(sig == NULL || rte_pktmbuf_headroom(m) < L3TypeLengthHeadroom + sizeofHeadroom)) {
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
  rte_memcpy(head, meta, sizeofMeta);
  head += sizeofMeta;
  *head++ = TtContent;
  head += TlvEncoder_WriteVarNum(head, contentL);

  return Encode_Finish(m);
}

__attribute__((nonnull)) static Packet*
Encode_Linear(DataGen* gen, LName prefix, PacketMempools* mp, uint16_t fragmentPayloadSize)
{
  uint32_t pktLen = L3TypeLengthHeadroom + L3TypeLengthHeadroom + // Data TL + Name TL
                    prefix.length + gen->tpl->pkt_len;
  uint32_t fragCount = SPDK_CEIL_DIV(pktLen, fragmentPayloadSize);
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

  TlvDecoder d = TlvDecoder_Init(gen->tpl);
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
