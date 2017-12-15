#include "interest-pkt.h"

NdnError
DecodeInterest(TlvDecoder* d, InterestPkt* interest, size_t* len)
{
  TlvElement interestEle;
  NdnError e = DecodeTlvElementExpectType(d, TT_Interest, &interestEle, len);
  RETURN_IF_UNLIKELY_ERROR;

  memset(interest, 0, sizeof(InterestPkt));
  interest->lifetime = DEFAULT_INTEREST_LIFETIME;

  TlvDecoder d1;
  TlvElement_MakeValueDecoder(&interestEle, &d1);
  size_t len1;

  e = DecodeName(&d1, &interest->name, &len1);
  RETURN_IF_UNLIKELY_ERROR;

  if (MbufLoc_PeekOctet(&d1) == TT_Selectors) {
    TlvElement selectorsEle;
    e = DecodeTlvElement(&d1, &selectorsEle, &len1);
    RETURN_IF_UNLIKELY_ERROR;

    TlvDecoder d2;
    TlvElement_MakeValueDecoder(&selectorsEle, &d2);
    while (!MbufLoc_IsEnd(&d2)) {
      TlvElement selectorEle;
      e = DecodeTlvElement(&d2, &selectorEle, &len1);
      RETURN_IF_UNLIKELY_ERROR;

      if (selectorEle.type != TT_MustBeFresh) {
        continue; // ignore unknown selector
      }
      interest->mustBeFresh = true;
    }
  }

  TlvElement nonceEle;
  e = DecodeTlvElementExpectType(&d1, TT_Nonce, &nonceEle, &len1);
  RETURN_IF_UNLIKELY_ERROR;
  if (unlikely(nonceEle.length != sizeof(uint32_t))) {
    return NdnError_BadNonceLength;
  }
  MbufLoc_Copy(&interest->nonce, &nonceEle.value);
  interest->nonce.rem = sizeof(uint32_t);

  if (MbufLoc_PeekOctet(&d1) == TT_InterestLifetime) {
    TlvElement lifetimeEle;
    e = DecodeTlvElement(&d1, &lifetimeEle, &len1);
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
    e = DecodeTlvElement(&d1, &fhEle, &len1);
    RETURN_IF_UNLIKELY_ERROR;

    TlvDecoder d2;
    TlvElement_MakeValueDecoder(&fhEle, &d2);
    for (int i = 0; i < INTEREST_MAX_FORWARDING_HINTS; ++i) {
      if (MbufLoc_IsEnd(&d2)) {
        break;
      }
      TlvElement delegationEle;
      e = DecodeTlvElementExpectType(&d2, TT_Delegation, &delegationEle, &len1);
      RETURN_IF_UNLIKELY_ERROR;

      TlvDecoder d3;
      TlvElement_MakeValueDecoder(&delegationEle, &d3);
      TlvElement preferenceEle;
      e = DecodeTlvElementExpectType(&d3, TT_Preference, &preferenceEle, &len1);
      RETURN_IF_UNLIKELY_ERROR;
      e = DecodeName(&d3, &interest->fwHints[i], &len1);
      RETURN_IF_UNLIKELY_ERROR;
      ++interest->nFwHints;
    }
  }

  return NdnError_OK;
}

void
InterestPkt_SetNonce(InterestPkt* interest, uint32_t nonce)
{
  // TODO
}