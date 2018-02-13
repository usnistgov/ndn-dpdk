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
    uint8_t selectorsT;
    uint8_t selectorsL;
    uint8_t mustBeFreshT;
    uint8_t mustBeFreshL;

    uint8_t nonceT;
    uint8_t nonceL;
    rte_be16_t nonceVhi;
    rte_be16_t nonceVlo;

    // InterestLifetime is a NonNegativeInteger fields, but NDN protocol does not
    // require NonNegativeInteger to use minimal length encoding.
    uint8_t interestLifetimeT;
    uint8_t interestLifetimeL;
    rte_be32_t interestLifetimeV;
  };
  struct Mid mid;
  static_assert(sizeof(mid) == 16, "");
  static_assert(sizeof(mid) - offsetof(struct Mid, nonceT) == 12, "");

  const uint8_t TT_Selectors = 0x09; // XXX
  mid.selectorsT = TT_Selectors;
  mid.selectorsL = 2;
  mid.mustBeFreshT = TT_MustBeFresh;
  mid.mustBeFreshL = 0;
  mid.nonceT = TT_Nonce;
  mid.nonceL = 4;
  int nonceRand = lrand48();
  mid.nonceVhi = nonceRand >> 16;
  mid.nonceVlo = nonceRand;
  mid.interestLifetimeT = TT_InterestLifetime;
  mid.interestLifetimeL = 4;
  mid.interestLifetimeV = rte_cpu_to_be_32(tpl->lifetime);

  int midOffset = ((int)!tpl->mustBeFresh) * offsetof(struct Mid, nonceT);
  int midSize = sizeof(mid) - midOffset;
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
