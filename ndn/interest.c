#include "interest.h"

NdnError
PInterest_FromPacket(PInterest* interest, struct rte_mbuf* pkt,
                     struct rte_mempool* nameMp)
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
  interest->name.v = TlvElement_LinearizeValue(&ele1, pkt, nameMp, &d1);
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
        TlvElement_LinearizeValue(&ele3, pkt, nameMp, &d3);
      RETURN_IF_UNLIKELY_NULL(interest->fh[i].value, NdnError_AllocError);
      interest->fh[i].length = ele3.length;
      ++interest->nFhs;
      MbufLoc_CopyPos(&d2, &d3);
    }
    MbufLoc_CopyPos(&d1, &d2);
    D1_NEXT;
  }

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
