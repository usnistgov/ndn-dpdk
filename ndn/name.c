#include "name.h"
#include <rte_per_lcore.h>

NdnError
DecodeName(TlvDecoder* d, Name* n)
{
  TlvElement nameEle;
  NdnError e = DecodeTlvElementExpectType(d, TT_Name, &nameEle);
  RETURN_IF_UNLIKELY_ERROR;

  if (unlikely(nameEle.length > NAME_MAX_LENGTH)) {
    return NdnError_NameTooLong;
  }

  n->nOctets = nameEle.length;
  n->nComps = 0;
  n->digestPos.m = NULL;

  TlvDecoder compsD;
  TlvElement_MakeValueDecoder(&nameEle, &compsD);

  while (!MbufLoc_IsEnd(&compsD)) {
    TlvElement compEle;
    e = DecodeTlvElement(&compsD, &compEle);
    RETURN_IF_UNLIKELY_ERROR;
    if (likely(n->nComps < NAME_MAX_INDEXED_COMPS)) {
      MbufLoc_Copy(&n->compPos[n->nComps], &compEle.first);
    }

    if (unlikely(compEle.type == TT_ImplicitSha256DigestComponent)) {
      if (compEle.length != 32) {
        return NdnError_BadDigestComponentLength;
      }
      MbufLoc_Copy(&n->digestPos, &compEle.first);
    } else if (unlikely(n->digestPos.m != NULL)) {
      return NdnError_NameHasComponentAfterDigest;
    }

    ++n->nComps;
  }

  return NdnError_OK;
}

#define LINEARIZE_BUFID_INTERNAL_START NAME_LINEARIZE_MAX_BUFID
#define LINEARIZE_BUFID_COMPARE_L (LINEARIZE_BUFID_INTERNAL_START + 0)
#define LINEARIZE_BUFID_COMPARE_R (LINEARIZE_BUFID_INTERNAL_START + 1)
#define LINEARIZE_BUFID_MAX (LINEARIZE_BUFID_INTERNAL_START + 2)

struct NameLinearizeBuffer
{
  uint8_t buf[LINEARIZE_BUFID_MAX][NAME_MAX_LENGTH];
};
RTE_DEFINE_PER_LCORE(struct NameLinearizeBuffer, nameLinearizeBuf);

const uint8_t*
Name_LinearizeComps(const Name* n, int bufid)
{
  assert(n->nOctets <= NAME_MAX_LENGTH);
  assert(bufid < LINEARIZE_BUFID_MAX);

  MbufLoc ml;
  MbufLoc_Copy(&ml, &n->compPos[0]);

  uint32_t nRead;
  const uint8_t* linearBuf = MbufLoc_Read(
    &ml, RTE_PER_LCORE(nameLinearizeBuf).buf[bufid], n->nOctets, &nRead);
  assert(nRead == n->nOctets);
  return linearBuf;
}

void
__Name_GetComp_PastIndexed(const Name* n, uint16_t i, TlvElement* ele)
{
  assert(n->nComps >= NAME_MAX_INDEXED_COMPS);
  assert(i >= NAME_MAX_INDEXED_COMPS);

  TlvDecoder d;
  uint16_t j = NAME_MAX_INDEXED_COMPS - 1;
  MbufLoc_Copy(&d, &n->compPos[j]);
  for (; j <= i; ++j) {
    NdnError e = DecodeTlvElement(&d, ele);
    assert(e == NdnError_OK); // cannot error in valid name
  }

  // last DecodeTlvElement invocation was on i-th element
}

NameCompareResult
Name_Compare(const Name* lhs, const Name* rhs)
{
  if (lhs->nComps == 0) {
    if (rhs->nComps == 0) {
      return NAMECMP_EQUAL;
    }
    return NAMECMP_LPREFIX;
  }
  if (rhs->nComps == 0) {
    return NAMECMP_RPREFIX;
  }

  const uint8_t* compsL = Name_LinearizeComps(lhs, LINEARIZE_BUFID_COMPARE_L);
  const uint8_t* compsR = Name_LinearizeComps(rhs, LINEARIZE_BUFID_COMPARE_R);

  uint16_t minOctets =
    lhs->nOctets <= rhs->nOctets ? lhs->nOctets : rhs->nOctets;
  int cmp = memcmp(compsL, compsR, minOctets);
  if (cmp != 0) {
    return ((cmp > 0) - (cmp < 0)) << 1;
  }
  cmp = lhs->nComps - rhs->nComps;
  return (cmp > 0) - (cmp < 0);
}