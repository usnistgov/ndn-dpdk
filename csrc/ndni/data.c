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
  TlvDecoder_New(&d, pkt);
  uint32_t length0, type0 = TlvDecoder_ReadTL(&d, &length0);
  NDNDPDK_ASSERT(type0 == TtData);

  TlvDecoder_EachTL (&d, type, length) {
    switch (type) {
      case TtName: {
        const uint8_t* v;
        if (unlikely(length > NameMaxLength || (v = TlvDecoder_Linearize(&d, length)) == NULL)) {
          return false;
        }
        LName lname = LName_Init(length, v);
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

static DataSatisfyResult
PData_CanSatisfy_HasDigestComp_(PData* data, PInterest* interest)
{
  LName nameI = PName_ToLName(&interest->name);
  LName nameD = PName_ToLName(&data->name);
  if (nameI.length != nameD.length + ImplicitDigestSize ||
      memcmp(nameI.value, nameD.value, nameD.length) != 0) {
    return DataSatisfyNo;
  }

  if (!data->hasDigest) {
    return DataSatisfyNeedDigest;
  }

  return memcmp(RTE_PTR_ADD(nameI.value, nameI.length - ImplicitDigestLength), data->digest,
                ImplicitDigestLength);
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
  return npkt;
}

Packet*
DataGen_Encode(DataGen* gen, struct rte_mbuf* seg0, struct rte_mbuf* seg1, LName prefix)
{
  struct rte_mbuf* tpl1 = (struct rte_mbuf*)gen;
  uint16_t nameSuffixL = tpl1->vlan_tci;

  NDNDPDK_ASSERT(RTE_MBUF_DIRECT(seg0) && rte_pktmbuf_is_contiguous(seg0) &&
                 rte_mbuf_refcnt_read(seg0) == 1 && seg0->data_len == 0 &&
                 seg0->buf_len >= DataGenDataroom);
  seg0->data_off = seg0->buf_len;
  if (likely(prefix.length > 0)) {
    rte_memcpy(rte_pktmbuf_prepend(seg0, prefix.length), prefix.value, prefix.length);
  }
  TlvEncoder_PrependTL(seg0, TtName, prefix.length + nameSuffixL);

  rte_pktmbuf_attach(seg1, tpl1);
  bool ok = Mbuf_Chain(seg0, seg0, seg1);
  NDNDPDK_ASSERT(ok);
  TlvEncoder_PrependTL(seg0, TtData, seg0->pkt_len);

  Packet* output = Packet_FromMbuf(seg0);
  Packet_SetType(output, PktSData);
  *Packet_GetLpL3Hdr(output) = (const LpL3){ 0 };
  return output;
}
