#include "name.h"
#include "tlv-decode.h"

static bool
IsValidNameComponentType(uint64_t type)
{
  return 1 <= type && type <= 32767;
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

    if (n->nComps > 0 && n->nComps <= NAME_N_CACHED_COMPS) {
      n->comp[n->nComps - 1] = off;
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
__PName_SeekCompStart(PName* n, const uint8_t* input, uint16_t i)
{
  assert(i > NAME_N_CACHED_COMPS);
  uint16_t off = n->comp[NAME_N_CACHED_COMPS - 1];
  for (uint16_t j = NAME_N_CACHED_COMPS; j < i; ++j) {
    uint64_t compT, compL;
    off += ParseTlvTypeLength(input + off, n->nOctets - off, &compT, &compL);
    off += compL;
  }
  return off;
}

uint64_t
LName_ComputeHash(LName n)
{
  SipHash h;
  SipHash_Init(&h, &theNameHashKey);
  SipHash_Write(&h, n.value, n.length);
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
