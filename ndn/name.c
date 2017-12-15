#include "name.h"
#include <rte_per_lcore.h>

struct NameCompareBuffer
{
  uint8_t buf[NAME_MAX_LENGTH];
};
RTE_DEFINE_PER_LCORE(struct NameCompareBuffer, nameCompBufA);
RTE_DEFINE_PER_LCORE(struct NameCompareBuffer, nameCompBufB);

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

  MbufLoc ml;
  MbufLoc_Copy(&ml, &lhs->compPos[0]);
  size_t nRead =
    MbufLoc_Read(&ml, RTE_PER_LCORE(nameCompBufA).buf, lhs->nOctets);
  assert(nRead == lhs->nOctets);
  MbufLoc_Copy(&ml, &rhs->compPos[0]);
  nRead = MbufLoc_Read(&ml, RTE_PER_LCORE(nameCompBufB).buf, rhs->nOctets);
  assert(nRead == rhs->nOctets);
  // TODO don't copy in MbufLoc_Read if whole name is in same segment

  uint16_t minOctets =
    lhs->nOctets <= rhs->nOctets ? lhs->nOctets : rhs->nOctets;
  int cmp = memcmp(RTE_PER_LCORE(nameCompBufA).buf,
                   RTE_PER_LCORE(nameCompBufB).buf, minOctets);
  if (cmp != 0) {
    return ((cmp > 0) - (cmp < 0)) << 1;
  }
  cmp = lhs->nComps - rhs->nComps;
  return (cmp > 0) - (cmp < 0);
}