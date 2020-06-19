#include "data.h"
#include "interest.h"
#include "packet.h"

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
