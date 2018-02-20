#include "interest.h"
#include "tlv-encoder.h"

NdnError
PInterest_FromPacket(PInterest* interest, struct rte_mbuf* pkt,
                     struct rte_mempool* mpName)
{
  TlvDecodePos d0;
  MbufLoc_Init(&d0, pkt);
  TlvElement interestEle;
  NdnError e = DecodeTlvElementExpectType(&d0, TT_Interest, &interestEle);
  RETURN_IF_UNLIKELY_ERROR;

  TlvDecodePos d1;
  TlvElement_MakeValueDecoder(&interestEle, &d1);
  TlvElement ele1;

#define D1_NEXT                                                                \
  do {                                                                         \
    if (MbufLoc_IsEnd(&d1)) {                                                  \
      return NdnError_OK;                                                      \
    }                                                                          \
    e = DecodeTlvElement(&d1, &ele1);                                          \
    RETURN_IF_UNLIKELY_ERROR;                                                  \
  } while (false)

  e = DecodeTlvElementExpectType(&d1, TT_Name, &ele1);
  RETURN_IF_UNLIKELY_ERROR;
  if (unlikely(ele1.length == 0)) {
    return NdnError_NameIsEmpty;
  }
  interest->name.v = TlvElement_LinearizeValue(&ele1, pkt, mpName, &d1);
  RETURN_IF_UNLIKELY_NULL(interest->name.v, NdnError_AllocError);
  e = PName_Parse(&interest->name.p, ele1.length, interest->name.v);
  RETURN_IF_UNLIKELY_ERROR;

  interest->nonce = 0;
  interest->lifetime = DEFAULT_INTEREST_LIFETIME;
  interest->hopLimit = HOP_LIMIT_OMITTED;
  interest->canBePrefix = false;
  interest->mustBeFresh = false;
  interest->nFhs = 0;
  interest->thisFhIndex = -1;

  D1_NEXT;
  if (ele1.type == TT_CanBePrefix) {
    interest->canBePrefix = true;
    D1_NEXT;
  }

  if (ele1.type == TT_MustBeFresh) {
    interest->mustBeFresh = true;
    D1_NEXT;
  }

  if (ele1.type == TT_ForwardingHint) {
    TlvDecodePos d2;
    TlvElement_MakeValueDecoder(&ele1, &d2);
    for (int i = 0; i < INTEREST_MAX_FHS; ++i) {
      if (MbufLoc_IsEnd(&d2)) {
        break;
      }
      TlvElement delegationEle;
      e = DecodeTlvElementExpectType(&d2, TT_Delegation, &delegationEle);
      RETURN_IF_UNLIKELY_ERROR;

      TlvDecodePos d3;
      TlvElement_MakeValueDecoder(&delegationEle, &d3);
      TlvElement ele3;
      e = DecodeTlvElementExpectType(&d3, TT_Preference, &ele3);
      RETURN_IF_UNLIKELY_ERROR;
      e = DecodeTlvElementExpectType(&d3, TT_Name, &ele3);
      interest->fh[i].value =
        TlvElement_LinearizeValue(&ele3, pkt, mpName, &d3);
      RETURN_IF_UNLIKELY_NULL(interest->fh[i].value, NdnError_AllocError);
      interest->fh[i].length = ele3.length;
      ++interest->nFhs;
      MbufLoc_CopyPos(&d2, &d3);
    }
    MbufLoc_CopyPos(&d1, &d2);
    D1_NEXT;
  }

  // mark position of Nonce and InterestLifetime, or where they may be inserted
  MbufLoc_Copy(&interest->guiderLoc, &ele1.first);

  if (ele1.type == TT_Nonce) {
    if (unlikely(ele1.length != 4)) {
      return NdnError_BadNonceLength;
    }
    // overwriting ele1.value, but it's okay because we don't need it later
    rte_le32_t nonceV;
    bool ok = MbufLoc_ReadU32(&ele1.value, &nonceV);
    assert(ok); // must succeed because length is checked
    interest->nonce = rte_le_to_cpu_32(nonceV);
    D1_NEXT;
  }

  if (ele1.type == TT_InterestLifetime) {
    uint64_t lifetimeV;
    bool ok = TlvElement_ReadNonNegativeInteger(&ele1, &lifetimeV);
    if (unlikely(!ok || lifetimeV >= UINT32_MAX)) {
      return NdnError_BadInterestLifetime;
    }
    interest->lifetime = (uint32_t)lifetimeV;
    D1_NEXT;
  }

  if (ele1.type == TT_HopLimit) {
    if (unlikely(ele1.length != 1)) {
      return NdnError_BadHopLimitLength;
    }
    const uint8_t* hopLimitV = TlvElement_GetLinearValue(&ele1);
    if (unlikely(*hopLimitV == 0)) {
      interest->hopLimit = HOP_LIMIT_ZERO;
    } else {
      interest->hopLimit = --(*(uint8_t*)hopLimitV);
    }
    D1_NEXT;
  }

  return NdnError_OK;
#undef D1_NEXT
}

NdnError
PInterest_ParseFh(PInterest* interest, uint8_t index)
{
  assert(index < interest->nFhs);
  if (interest->thisFhIndex == index) {
    return NdnError_OK;
  }

  NdnError e = PName_Parse(&interest->thisFh.p, interest->fh[index].length,
                           interest->fh[index].value);
  RETURN_IF_UNLIKELY_ERROR;

  interest->thisFh.v = interest->fh[index].value;
  interest->thisFhIndex = index;
  return NdnError_OK;
}

uint16_t
__InterestTemplate_Prepare(InterestTemplate* tpl, uint8_t* buffer,
                           uint16_t bufferSize, const uint8_t* fhV)
{
  tpl->bufferOff = 0;
  uint16_t size = 0;
  if (tpl->canBePrefix) {
    size += SizeofVarNum(TT_CanBePrefix) + SizeofVarNum(0);
  }
  if (tpl->mustBeFresh) {
    size += SizeofVarNum(TT_MustBeFresh) + SizeofVarNum(0);
  }
  if (tpl->fhL > 0) {
    size += SizeofVarNum(TT_ForwardingHint) + SizeofVarNum(tpl->fhL) + tpl->fhL;
  }
  {
    size += SizeofVarNum(TT_Nonce) + SizeofVarNum(4);
    tpl->nonceOff = size;
    while (tpl->nonceOff % 4 != 0) {
      ++tpl->bufferOff;
      ++tpl->nonceOff;
    }
    size += 4;
  }
  if (tpl->lifetime != DEFAULT_INTEREST_LIFETIME) {
    size += SizeofVarNum(TT_InterestLifetime) + SizeofVarNum(4) + 4;
  }
  if (tpl->hopLimit != HOP_LIMIT_OMITTED) {
    size += SizeofVarNum(TT_HopLimit) + SizeofVarNum(1) + 1;
  }
  if (size > bufferSize) {
    return size;
  }

  uint8_t* p = buffer + tpl->bufferOff;
  if (tpl->canBePrefix) {
    p = EncodeVarNum(p, TT_CanBePrefix);
    p = EncodeVarNum(p, 0);
  }
  if (tpl->mustBeFresh) {
    p = EncodeVarNum(p, TT_MustBeFresh);
    p = EncodeVarNum(p, 0);
  }
  if (tpl->fhL > 0) {
    p = EncodeVarNum(p, TT_ForwardingHint);
    p = EncodeVarNum(p, tpl->fhL);
    rte_memcpy(p, fhV, tpl->fhL);
    p += tpl->fhL;
  }
  {
    p = EncodeVarNum(p, TT_Nonce);
    p = EncodeVarNum(p, 4);
    assert(p == buffer + tpl->nonceOff);
    p += 4;
  }
  if (tpl->lifetime != DEFAULT_INTEREST_LIFETIME) {
    p = EncodeVarNum(p, TT_InterestLifetime);
    p = EncodeVarNum(p, 4);
    rte_be32_t lifetimeV = rte_cpu_to_be_32(tpl->lifetime);
    rte_memcpy(p, &lifetimeV, 4);
    p += 4;
  }
  if (tpl->hopLimit != HOP_LIMIT_OMITTED) {
    p = EncodeVarNum(p, TT_HopLimit);
    p = EncodeVarNum(p, 1);
    *p++ = (uint8_t)tpl->hopLimit;
  }
  assert(p == buffer + tpl->bufferOff + size);
  tpl->bufferSize = size;
  return 0;
}

void
__EncodeInterest(struct rte_mbuf* m, const InterestTemplate* tpl,
                 uint8_t* preparedBuffer, uint16_t nameSuffixL,
                 const uint8_t* nameSuffixV, uint16_t paramL,
                 const uint8_t* paramV, const uint8_t* namePrefixV)
{
  assert(rte_pktmbuf_headroom(m) >= EncodeInterest_GetHeadroom());
  assert(rte_pktmbuf_tailroom(m) >=
         EncodeInterest_GetTailroom(tpl, nameSuffixL, paramL));
  TlvEncoder* en = MakeTlvEncoder(m);

  AppendVarNum(en, TT_Name);
  AppendVarNum(en, tpl->namePrefix.length + nameSuffixL);
  if (likely(tpl->namePrefix.length > 0)) {
    rte_memcpy(rte_pktmbuf_append(m, tpl->namePrefix.length), namePrefixV,
               tpl->namePrefix.length);
  }
  if (likely(nameSuffixL > 0)) {
    rte_memcpy(rte_pktmbuf_append(m, nameSuffixL), nameSuffixV, nameSuffixL);
  }

  uint32_t* nonce = (uint32_t*)(preparedBuffer + tpl->nonceOff);
  *nonce = (uint32_t)lrand48();
  rte_memcpy(rte_pktmbuf_append(m, tpl->bufferSize),
             preparedBuffer + tpl->bufferOff, tpl->bufferSize);

  if (paramL > 0) {
    AppendVarNum(en, TT_Parameters);
    AppendVarNum(en, paramL);
    rte_memcpy(rte_pktmbuf_append(m, paramL), paramV, paramL);
  }

  PrependVarNum(en, m->pkt_len);
  PrependVarNum(en, TT_Interest);
}
