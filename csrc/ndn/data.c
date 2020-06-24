#include "data.h"
#include "interest.h"
#include "packet.h"
#include "tlv-encoder.h"

// clang-format off
static const uint8_t FAKESIG[] = {
  TtDSigInfo, 0x03,
    TtSigType, 0x01, SigHmacWithSha256,
  TtDSigValue, 0x20,
    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
};
// clang-format on

NdnError
PData_FromPacket(PData* data, struct rte_mbuf* pkt, struct rte_mempool* nameMp)
{
  MbufLoc d0;
  MbufLoc_Init(&d0, pkt);
  TlvElement dataEle;
  NdnError e = TlvElement_Decode(&dataEle, &d0, TtData);
  RETURN_IF_ERROR;
  data->size = dataEle.size;

  MbufLoc d1;
  TlvElement_MakeValueDecoder(&dataEle, &d1);

  TlvElement nameEle;
  e = TlvElement_Decode(&nameEle, &d1, TtName);
  RETURN_IF_ERROR;
  if (unlikely(nameEle.length == 0)) {
    data->name.v = NULL;
    PName_Clear(&data->name.p);
  } else {
    data->name.v = TlvElement_LinearizeValue(&nameEle, pkt, nameMp, &d1);
    RETURN_IF_NULL(data->name.v, NdnErrAllocError);
    e = PName_Parse(&data->name.p, nameEle.length, data->name.v);
    RETURN_IF_ERROR;
  }

  data->freshnessPeriod = 0;
  TlvElement metaEle;
  e = TlvElement_Decode(&metaEle, &d1, TtMetaInfo);
  if (e == NdnErrIncomplete || e == NdnErrBadType) {
    return NdnErrOK; // MetaInfo not present
  }
  RETURN_IF_ERROR;

  MbufLoc d2;
  TlvElement_MakeValueDecoder(&metaEle, &d2);
  while (!MbufLoc_IsEnd(&d2)) {
    TlvElement metaChild;
    e = TlvElement_Decode(&metaChild, &d2, TtInvalid);
    RETURN_IF_ERROR;

    if (metaChild.type != TtFreshnessPeriod) {
      continue; // ignore other children of MetaInfo
    }

    uint64_t fpV;
    e = TlvElement_ReadNonNegativeInteger(&metaChild, &fpV);
    RETURN_IF_ERROR;
    data->freshnessPeriod = (uint32_t)RTE_MIN(UINT32_MAX, fpV);
    break;
  }

  return NdnErrOK;
}

DataSatisfyResult
PData_CanSatisfy(PData* data, PInterest* interest)
{
  if (unlikely(interest->mustBeFresh && data->freshnessPeriod == 0)) {
    return DataSatisfyNo;
  }

  const LName* interestLName = (const LName*)&interest->name;
  const LName* dataLName = (const LName*)&data->name;
  NameCompareResult cmp = LName_Compare(*interestLName, *dataLName);

  if (unlikely(interest->name.p.hasDigestComp)) {
    if (cmp != NAMECMP_RPREFIX ||
        interest->name.p.nComps != data->name.p.nComps + 1) {
      return DataSatisfyNo;
    }

    if (!data->hasDigest) {
      return DataSatisfyNeedDigest;
    }

    NameComp digestComp = Name_GetComp(&interest->name, data->name.p.nComps);
    assert(digestComp.size == 34);
    const uint8_t* digest = RTE_PTR_ADD(digestComp.tlv, 2);
    return memcmp(digest, data->digest, 32) == 0 ? DataSatisfyYes
                                                 : DataSatisfyNo;
  }

  return (cmp == NAMECMP_EQUAL ||
          (interest->canBePrefix && cmp == NAMECMP_LPREFIX))
           ? DataSatisfyYes
           : DataSatisfyNo;
}

void
DataDigest_Prepare(Packet* npkt, struct rte_crypto_op* op)
{
  PData* data = Packet_GetDataHdr(npkt);
  struct rte_mbuf* pkt = Packet_ToMbuf(npkt);
  CryptoOp_PrepareSha256Digest(op, pkt, 0, data->size, data->digest);
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

DataGen*
DataGen_New(struct rte_mbuf* m,
            uint16_t nameSuffixL,
            const uint8_t* nameSuffixV,
            uint32_t freshnessPeriod,
            uint16_t contentL,
            const uint8_t* contentV)
{
  TlvEncoder* en = MakeTlvEncoder(m);
  if (nameSuffixL > 0) {
    rte_memcpy(rte_pktmbuf_append(m, nameSuffixL), nameSuffixV, nameSuffixL);
  }

  if (freshnessPeriod != 0) {
    typedef struct MetaInfoF
    {
      uint8_t metaInfoT;
      uint8_t metaInfoL;
      uint8_t freshnessPeriodT;
      uint8_t freshnessPeriodL;
      rte_be32_t freshnessPeriodV;
    } __rte_packed MetaInfoF;

    MetaInfoF* f = (MetaInfoF*)TlvEncoder_Append(en, sizeof(MetaInfoF));
    f->metaInfoT = TtMetaInfo;
    f->metaInfoL = 6;
    f->freshnessPeriodT = TtFreshnessPeriod;
    f->freshnessPeriodL = 4;
    *(unaligned_uint32_t*)&f->freshnessPeriodV =
      rte_cpu_to_be_32(freshnessPeriod);
  }

  if (contentL != 0) {
    AppendVarNum(en, TtContent);
    AppendVarNum(en, contentL);
    rte_memcpy(rte_pktmbuf_append(m, contentL), contentV, contentL);
  }

  rte_memcpy(rte_pktmbuf_append(m, sizeof(FAKESIG)), FAKESIG, sizeof(FAKESIG));

  m->vlan_tci = nameSuffixL;
  return (DataGen*)m;
}

void
DataGen_Close(DataGen* gen)
{
  rte_pktmbuf_free((struct rte_mbuf*)gen);
}

void
DataGen_Encode_(DataGen* gen,
                struct rte_mbuf* seg0,
                struct rte_mbuf* seg1,
                uint16_t namePrefixL,
                const uint8_t* namePrefixV)
{
  struct rte_mbuf* tailTpl = (struct rte_mbuf*)gen;
  uint16_t nameSuffixL = tailTpl->vlan_tci;
  rte_pktmbuf_attach(seg1, tailTpl);

  TlvEncoder* en = MakeTlvEncoder(seg0);
  AppendVarNum(en, TtName);
  AppendVarNum(en, namePrefixL + nameSuffixL);
  if (likely(namePrefixL > 0)) {
    rte_memcpy(rte_pktmbuf_append(seg0, namePrefixL), namePrefixV, namePrefixL);
  }

  rte_pktmbuf_chain(seg0, seg1);
  PrependVarNum(en, seg0->pkt_len);
  PrependVarNum(en, TtData);
}
