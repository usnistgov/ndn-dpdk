#include "data.h"
#include "packet.h"
#include "tlv-decoder.h"
#include "tlv-encoder.h"

// helperScratch should be small enough not to increase PacketPriv size
static_assert(sizeof(PData) <= sizeof(PInterest), "");

static struct {
  unaligned_uint16_t sigInfoTL;
  unaligned_uint16_t sigTypeTL;
  uint8_t sigTypeV;
  unaligned_uint16_t sigValueTL;
} __rte_packed NullSig;

RTE_INIT(InitNullSig) {
  NullSig.sigInfoTL =
    TlvEncoder_ConstTL1(TtDSigInfo, sizeof(NullSig.sigTypeTL) + sizeof(NullSig.sigTypeV));
  NullSig.sigTypeTL = TlvEncoder_ConstTL1(TtSigType, sizeof(NullSig.sigTypeV));
  NullSig.sigTypeV = SigNull;
  NullSig.sigValueTL = TlvEncoder_ConstTL1(TtDSigValue, 0);

  static_assert(sizeof(NullSig) == DataEncNullSigLen, "");
}

uint8_t DataEnc_NoMetaInfo[] = {0};

__attribute__((nonnull)) static inline bool
PData_ParseMetaInfo(PData* data, TlvDecoder* d, ParseFor parseFor) {
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
PData_Parse(PData* data, struct rte_mbuf* pkt, ParseFor parseFor) {
  NDNDPDK_ASSERT(RTE_MBUF_DIRECT(pkt) && rte_mbuf_refcnt_read(pkt) == 1);
  *data = (const PData){0};

  TlvDecoder d = TlvDecoder_Init(pkt);
  uint32_t length0, type0 = TlvDecoder_ReadTL(&d, &length0);
  NDNDPDK_ASSERT(type0 == TtData);

  TlvDecoder_EachTL (&d, type, length) {
    switch (type) {
      case TtName: {
        LName lname = (LName){.length = length};
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
PData_CanSatisfy_HasDigestComp_(PData* data, PInterest* interest) {
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
PData_CanSatisfy(PData* data, PInterest* interest) {
  if (unlikely(interest->name.hasDigestComp)) {
    return PData_CanSatisfy_HasDigestComp_(data, interest);
  }

  int cmp = LName_IsPrefix(PName_ToLName(&interest->name), PName_ToLName(&data->name));
  return (interest->canBePrefix ? cmp >= 0 : cmp == 0) ? DataSatisfyYes : DataSatisfyNo;
}

struct rte_crypto_op*
DataDigest_Prepare(CryptoQueuePair* cqp, Packet* npkt) {
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
DataDigest_Enqueue(CryptoQueuePair* cqp, struct rte_crypto_op** ops, uint16_t count) {
  if (unlikely(count == 0)) {
    return 0;
  }
  uint16_t nEnq = rte_cryptodev_enqueue_burst(cqp->dev, cqp->qp, ops, count);
  return count - nEnq;
}

bool
DataDigest_Finish(struct rte_crypto_op* op, Packet** npkt) {
  NDNDPDK_ASSERT(op->mempool == NULL);
  *npkt = Packet_FromMbuf(op->sym->m_src);
  PData* data = Packet_GetDataHdr(*npkt);
  data->hasDigest = op->status == RTE_CRYPTO_OP_STATUS_SUCCESS;
  return data->hasDigest;
}

void
DataEnc_PrepareMetaInfo(uint8_t* room, ContentType ct, uint32_t freshness, LName finalBlock) {
  room[0] = TtMetaInfo;
  room[1] = 0;
#define APPEND(ptr, extraLength)                                                                   \
  do {                                                                                             \
    ptr = RTE_PTR_ADD(room, 2 + room[1]);                                                          \
    room[1] += sizeof(*ptr) + (extraLength);                                                       \
  } while (false)

  if (unlikely(ct != ContentBlob)) {
    struct ContentTypeF {
      unaligned_uint16_t contentTypeTL;
      uint8_t contentTypeV;
    } __rte_packed* f = NULL;
    APPEND(f, 0);
    f->contentTypeTL = TlvEncoder_ConstTL1(TtContentType, sizeof(f->contentTypeV));
    f->contentTypeV = ct;
  }

  if (freshness > 0) {
    struct FreshnessF {
      unaligned_uint16_t freshnessTL;
      unaligned_uint32_t freshnessV;
    } __rte_packed* f = NULL;
    APPEND(f, 0);
    f->freshnessTL = TlvEncoder_ConstTL1(TtFreshnessPeriod, sizeof(f->freshnessV));
    f->freshnessV = rte_cpu_to_be_32(freshness);
  }

  if (finalBlock.length > 0) {
    struct FinalBlockF {
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

  if (room[1] == 0) {
    room[0] = 0;
  }
}

__attribute__((nonnull)) static struct rte_mbuf*
DataEnc_EncodeCommon(LName prefix, LName suffix, const uint8_t* meta, uint32_t contentL,
                     bool allocContentL, struct iovec* iov, int* iovcnt, PacketMempools* mp,
                     uint16_t dataLen) {
  uint16_t sizeofMeta = DataEnc_SizeofMetaInfo(meta);
  uint8_t nameTL[L3TypeLengthHeadroom] = {TtName};
  uint16_t sizeofNameTL = 1 + TlvEncoder_WriteVarNum(&nameTL[1], prefix.length + suffix.length);
  uint8_t contentTL[L3TypeLengthHeadroom] = {TtContent};
  uint16_t sizeofContentTL = 1 + TlvEncoder_WriteVarNum(&contentTL[1], contentL);

  *iovcnt = LpMaxFragments;
  struct rte_mbuf* pkt = Mbuf_AllocRoom(
    mp->packet, iov, iovcnt, RTE_PKTMBUF_HEADROOM + LpHeaderHeadroom + L3TypeLengthHeadroom,
    dataLen == 0 ? 0 : dataLen - L3TypeLengthHeadroom, RTE_PKTMBUF_HEADROOM + LpHeaderHeadroom,
    dataLen,
    sizeofNameTL + prefix.length + suffix.length + sizeofMeta + sizeofContentTL +
      contentL * (int)allocContentL);
  if (unlikely(pkt == NULL)) {
    return NULL;
  }

  struct spdk_iov_xfer ix;
  spdk_iov_xfer_init(&ix, iov, *iovcnt);
  spdk_iov_xfer_from_buf(&ix, nameTL, sizeofNameTL);
  spdk_iov_xfer_from_buf(&ix, prefix.value, prefix.length);
  spdk_iov_xfer_from_buf(&ix, suffix.value, suffix.length);
  spdk_iov_xfer_from_buf(&ix, meta, sizeofMeta);
  spdk_iov_xfer_from_buf(&ix, contentTL, sizeofContentTL);
  Mbuf_RemainingIovec(ix, iov, iovcnt);
  return pkt;
}

__attribute__((nonnull)) static struct rte_mbuf*
DataEnc_EncodeLinear(LName prefix, LName suffix, const uint8_t* meta, uint32_t roomL,
                     struct iovec* roomIov, int* roomIovcnt, PacketMempools* mp,
                     uint16_t fragmentPayloadSize) {
  return DataEnc_EncodeCommon(prefix, suffix, meta, roomL, true, roomIov, roomIovcnt, mp,
                              fragmentPayloadSize);
}

__attribute__((nonnull)) static struct rte_mbuf*
DataEnc_EncodeChained(LName prefix, LName suffix, const uint8_t* meta, struct rte_mbuf* tplV,
                      PacketMempools* mp) {
  struct iovec iov[LpMaxFragments];
  int iovcnt = RTE_DIM(iov);
  struct rte_mbuf* pkt =
    DataEnc_EncodeCommon(prefix, suffix, meta, tplV->pkt_len, false, iov, &iovcnt, mp, 0);
  if (unlikely(pkt == NULL)) {
    return NULL;
  }
  NDNDPDK_ASSERT(iovcnt == 0);

  struct rte_mbuf* content = rte_pktmbuf_clone(tplV, mp->indirect);
  if (unlikely(content == NULL)) {
    rte_pktmbuf_free(pkt);
    return NULL;
  }

  int res = rte_pktmbuf_chain(pkt, content);
  if (unlikely(res != 0)) {
    rte_pktmbuf_free(pkt);
    rte_pktmbuf_free(content);
    return NULL;
  }

  return pkt;
}

struct rte_mbuf*
DataEnc_EncodeTpl(LName prefix, LName suffix, const uint8_t* meta, struct rte_mbuf* tplV,
                  struct iovec* tplIov, int tplIovcnt, PacketMempools* mp, PacketTxAlign align) {
  if (!align.linearize) {
    return DataEnc_EncodeChained(prefix, suffix, meta, tplV, mp);
  }

  struct iovec roomIov[LpMaxFragments];
  int roomIovcnt = 0;
  struct rte_mbuf* pkt = DataEnc_EncodeLinear(prefix, suffix, meta, tplV->pkt_len, roomIov,
                                              &roomIovcnt, mp, align.fragmentPayloadSize);
  if (unlikely(pkt == NULL)) {
    return NULL;
  }

  size_t nCopiedOctets = spdk_iovcpy(tplIov, tplIovcnt, roomIov, roomIovcnt);
  if (unlikely(nCopiedOctets != tplV->pkt_len)) {
    rte_pktmbuf_free(pkt);
    return NULL;
  }
  return pkt;
}

struct rte_mbuf*
DataEnc_EncodeRoom(LName prefix, LName suffix, const uint8_t* meta, uint32_t roomL,
                   struct iovec* roomIov, int* roomIovcnt, PacketMempools* mp,
                   PacketTxAlign align) {
  if (!align.linearize) {
    align.fragmentPayloadSize =
      rte_pktmbuf_data_room_size(mp->packet) - RTE_PKTMBUF_HEADROOM - LpHeaderHeadroom;
  }
  return DataEnc_EncodeLinear(prefix, suffix, meta, roomL, roomIov, roomIovcnt, mp,
                              align.fragmentPayloadSize);
}

__attribute__((nonnull)) static inline struct rte_mbuf*
DataEnc_SignChain(struct rte_mbuf* pkt, struct rte_mbuf* tail, PacketMempools* mp) {
  struct rte_mbuf* sigSeg = rte_pktmbuf_alloc(mp->packet);
  if (unlikely(sigSeg == NULL)) {
    return NULL;
  }
  sigSeg->data_off = RTE_PKTMBUF_HEADROOM + LpHeaderHeadroom;

  if (unlikely(!Mbuf_Chain(pkt, tail, sigSeg))) {
    rte_pktmbuf_free(sigSeg);
    return NULL;
  }
  return sigSeg;
}

__attribute__((nonnull)) static inline struct rte_mbuf*
DataEnc_SignDirect(struct rte_mbuf* pkt, struct rte_mbuf* tail, PacketMempools* mp,
                   uint16_t fragmentPayloadSize) {
  if (unlikely(tail->data_len + DataEncNullSigLen > fragmentPayloadSize ||
               rte_pktmbuf_tailroom(tail) < DataEncNullSigLen)) {
    return DataEnc_SignChain(pkt, tail, mp);
  }
  return tail;
}

Packet*
DataEnc_Sign(struct rte_mbuf* pkt, PacketMempools* mp, PacketTxAlign align) {
  struct rte_mbuf* tail = rte_pktmbuf_lastseg(pkt);
  if (align.linearize) {
    NDNDPDK_ASSERT(RTE_MBUF_DIRECT(tail) && rte_mbuf_refcnt_read(tail) == 1);
    tail = DataEnc_SignDirect(pkt, tail, mp, align.fragmentPayloadSize);
  } else if (RTE_MBUF_DIRECT(tail) && rte_mbuf_refcnt_read(tail) == 1) {
    tail = DataEnc_SignDirect(pkt, tail, mp, UINT16_MAX);
  } else {
    tail = DataEnc_SignChain(pkt, tail, mp);
  }

  if (unlikely(tail == NULL)) {
    rte_pktmbuf_free(pkt);
    return NULL;
  }

  rte_memcpy(rte_pktmbuf_mtod_offset(tail, void*, tail->data_len), &NullSig, DataEncNullSigLen);
  tail->data_len += DataEncNullSigLen;
  pkt->pkt_len += DataEncNullSigLen;
  return Packet_EncodeFinish_(pkt, TtData, PktSData);
}
