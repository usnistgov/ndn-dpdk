#include "interest-pkt.h"
#include "tlv-encoder.h"

NdnError
DecodeInterest(TlvDecoder* d, InterestPkt* interest)
{
  TlvElement interestEle;
  NdnError e = DecodeTlvElementExpectType(d, TT_Interest, &interestEle);
  RETURN_IF_UNLIKELY_ERROR;

  memset(interest, 0, sizeof(InterestPkt));
  interest->lifetime = DEFAULT_INTEREST_LIFETIME;

  TlvDecoder d1;
  TlvElement_MakeValueDecoder(&interestEle, &d1);

  e = DecodeName(&d1, &interest->name);
  RETURN_IF_UNLIKELY_ERROR;

  if (MbufLoc_PeekOctet(&d1) == TT_Selectors) {
    TlvElement selectorsEle;
    e = DecodeTlvElement(&d1, &selectorsEle);
    RETURN_IF_UNLIKELY_ERROR;

    TlvDecoder d2;
    TlvElement_MakeValueDecoder(&selectorsEle, &d2);
    while (!MbufLoc_IsEnd(&d2)) {
      TlvElement selectorEle;
      e = DecodeTlvElement(&d2, &selectorEle);
      RETURN_IF_UNLIKELY_ERROR;

      if (selectorEle.type != TT_MustBeFresh) {
        continue; // ignore unknown selector
      }
      interest->mustBeFresh = true;
      break;
    }
  }

  TlvElement nonceEle;
  e = DecodeTlvElementExpectType(&d1, TT_Nonce, &nonceEle);
  RETURN_IF_UNLIKELY_ERROR;
  if (unlikely(nonceEle.length != sizeof(uint32_t))) {
    return NdnError_BadNonceLength;
  }
  TlvElement_MakeValueDecoder(&nonceEle, &interest->nonce);

  if (MbufLoc_PeekOctet(&d1) == TT_InterestLifetime) {
    TlvElement lifetimeEle;
    e = DecodeTlvElement(&d1, &lifetimeEle);
    RETURN_IF_UNLIKELY_ERROR;

    uint64_t lifetimeVal;
    bool ok = TlvElement_ReadNonNegativeInteger(&lifetimeEle, &lifetimeVal);
    if (unlikely(!ok) || lifetimeVal >= UINT32_MAX) {
      return NdnError_BadInterestLifetime;
    }
    interest->lifetime = (uint32_t)lifetimeVal;
  }

  if (MbufLoc_PeekOctet(&d1) == TT_ForwardingHint) {
    TlvElement fhEle;
    e = DecodeTlvElement(&d1, &fhEle);
    RETURN_IF_UNLIKELY_ERROR;

    TlvDecoder d2;
    TlvElement_MakeValueDecoder(&fhEle, &d2);
    for (int i = 0; i < INTEREST_MAX_FORWARDING_HINTS; ++i) {
      if (MbufLoc_IsEnd(&d2)) {
        break;
      }
      TlvElement delegationEle;
      e = DecodeTlvElementExpectType(&d2, TT_Delegation, &delegationEle);
      RETURN_IF_UNLIKELY_ERROR;

      TlvDecoder d3;
      TlvElement_MakeValueDecoder(&delegationEle, &d3);
      TlvElement preferenceEle;
      e = DecodeTlvElementExpectType(&d3, TT_Preference, &preferenceEle);
      RETURN_IF_UNLIKELY_ERROR;
      e = DecodeName(&d3, &interest->fwHints[i]);
      RETURN_IF_UNLIKELY_ERROR;
      ++interest->nFwHints;
    }
  }

  return NdnError_OK;
}

void
InterestPkt_SetNonce(InterestPkt* interest, uint32_t nonce)
{
  assert(false);
  // TODO
}

void
__EncodeInterest(struct rte_mbuf* m, const InterestTemplate* tpl,
                 const uint8_t* namePrefix, const uint8_t* nameSuffix,
                 const uint8_t* fwHints)
{
  assert(rte_pktmbuf_headroom(m) >= EncodeInterest_GetHeadroom());
  assert(rte_pktmbuf_tailroom(m) >= EncodeInterest_GetTailroom(tpl));

  TlvEncoder* en = MakeTlvEncoder(m);

  AppendVarNum(en, TT_Name);
  AppendVarNum(en, tpl->namePrefixSize + tpl->nameSuffixSize);
  if (likely(tpl->namePrefixSize > 0)) {
    rte_memcpy(rte_pktmbuf_append(m, tpl->namePrefixSize), namePrefix,
               tpl->namePrefixSize);
  }
  if (likely(tpl->nameSuffixSize > 0)) {
    rte_memcpy(rte_pktmbuf_append(m, tpl->nameSuffixSize), nameSuffix,
               tpl->nameSuffixSize);
  }

  struct Mid
  {
    char _padding[2]; // make interestLifetimeV aligned

    uint8_t selectorsT;
    uint8_t selectorsL;
    uint8_t mustBeFreshT;
    uint8_t mustBeFreshL;

    // InterestLifetime is a NonNegativeInteger fields, but NDN protocol does not
    // require NonNegativeInteger to use minimal length encoding.
    uint8_t interestLifetimeT;
    uint8_t interestLifetimeL;
    rte_be32_t interestLifetimeV;

    uint8_t nonceT;
    uint8_t nonceL;
    rte_be16_t nonceVhi;
    rte_be16_t nonceVlo;

    char _end[0];
  };
  struct Mid mid;
  static_assert(
    offsetof(struct Mid, _end) - offsetof(struct Mid, selectorsT) == 16, "");
  static_assert(
    offsetof(struct Mid, _end) - offsetof(struct Mid, interestLifetimeT) == 12,
    "");

  mid.selectorsT = TT_Selectors;
  mid.selectorsL = 2;
  mid.mustBeFreshT = TT_MustBeFresh;
  mid.mustBeFreshL = 0;
  mid.interestLifetimeT = TT_InterestLifetime;
  mid.interestLifetimeL = 4;
  mid.interestLifetimeV = rte_cpu_to_be_32(tpl->lifetime);
  mid.nonceT = TT_Nonce;
  mid.nonceL = 4;
  int nonceRand = lrand48();
  mid.nonceVhi = nonceRand >> 16;
  mid.nonceVlo = nonceRand;

  int midOffset =
    offsetof(struct Mid, interestLifetimeT) -
    (int)tpl->mustBeFresh * (offsetof(struct Mid, interestLifetimeT) -
                             offsetof(struct Mid, selectorsT));
  int midSize = offsetof(struct Mid, _end) - midOffset;
  rte_memcpy(rte_pktmbuf_append(m, midSize), ((uint8_t*)&mid) + midOffset,
             midSize);

  if (tpl->fwHintsSize > 0) {
    AppendVarNum(en, TT_ForwardingHint);
    AppendVarNum(en, tpl->fwHintsSize);
    rte_memcpy(rte_pktmbuf_append(m, tpl->fwHintsSize), fwHints,
               tpl->fwHintsSize);
  }

  PrependVarNum(en, m->pkt_len);
  PrependVarNum(en, TT_Interest);
}
