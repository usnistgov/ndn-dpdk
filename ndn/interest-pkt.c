#include "interest-pkt.h"

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