#include "name.h"
#include "tlv-decode.h"

uint64_t
__LName_ComputeHash(uint16_t length, const uint8_t* value)
{
  SipHash h;
  SipHash_Init(&h, &theNameHashKey);
  SipHash_Write(&h, value, length);
  return SipHash_Final(&h);
}

NameCompareResult
LName_Compare(LName lhs, LName rhs)
{
  uint16_t minOctets = lhs.length <= rhs.length ? lhs.length : rhs.length;
  int cmp = memcmp(lhs.value, rhs.value, minOctets);
  if (cmp != 0) {
    return ((cmp > 0) - (cmp < 0)) << 1;
  }
  cmp = lhs.length - rhs.length;
  return (cmp > 0) - (cmp < 0);
}

static bool
IsValidNameComponentType(uint64_t type)
{
  return 1 <= type && type <= 65535;
}

NdnError
PName_Parse(PName* n, uint32_t length, const uint8_t* value)
{
  if (unlikely(length > NAME_MAX_LENGTH)) {
    return NdnError_NameTooLong;
  }

  n->nOctets = length;
  n->nComps = 0;
  n->hasDigestComp = false;
  n->hasHashes = false;

  uint32_t off = 0;
  while (off < length) {
    uint64_t compT, compL;
    uint32_t sizeofTL =
      ParseTlvTypeLength(value + off, length - off, &compT, &compL);
    if (unlikely(sizeofTL) == 0) {
      return NdnError_Incomplete;
    }
    uint64_t end = off + sizeofTL + compL;
    if (unlikely(end > length)) {
      return NdnError_Incomplete;
    }

    if (unlikely(!IsValidNameComponentType(compT))) {
      return NdnError_BadNameComponentType;
    }

    if (unlikely(compT == TT_ImplicitSha256DigestComponent)) {
      if (unlikely(compL != 32)) {
        return NdnError_BadDigestComponentLength;
      }
      n->hasDigestComp = true;
    }

    if (likely(n->nComps < PNAME_N_CACHED_COMPS)) {
      n->comp[n->nComps] = end;
    }

    ++n->nComps;
    off = end;
  }

  return NdnError_OK;
}

NdnError
PName_FromElement(PName* n, const TlvElement* ele)
{
  assert(ele->type == TT_Name);
  if (unlikely(!TlvElement_IsValueLinear(ele))) {
    return NdnError_Fragmented;
  }
  return PName_Parse(n, ele->length, TlvElement_GetLinearValue(ele));
}

uint16_t
__PName_SeekCompEnd(const PName* n, const uint8_t* input, uint16_t i)
{
  assert(i >= PNAME_N_CACHED_COMPS);
  uint16_t off = n->comp[PNAME_N_CACHED_COMPS - 1];
  for (uint16_t j = PNAME_N_CACHED_COMPS - 1; j < i; ++j) {
    uint64_t compT, compL;
    off += ParseTlvTypeLength(input + off, n->nOctets - off, &compT, &compL);
    off += compL;
  }
  return off;
}

void
__PName_HashToCache(PName* n, const uint8_t* input)
{
  SipHash h;
  SipHash_Init(&h, &theNameHashKey);

  uint16_t off = 0;
  for (uint16_t i = 0, last = n->nComps < PNAME_N_CACHED_COMPS
                                ? n->nComps
                                : PNAME_N_CACHED_COMPS;
       i < last; ++i) {
    SipHash_Write(&h, input + off, n->comp[i] - off);
    n->hash[i] = SipHash_Sum(&h);
    off = n->comp[i];
  }

  n->hasHashes = true;
}
