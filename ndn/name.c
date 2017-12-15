#include "name.h"

NdnError
DecodeName(TlvDecoder* d, Name* n, size_t* len)
{
  TlvElement nameEle;
  NdnError e = DecodeTlvElementExpectType(d, TT_Name, &nameEle, len);
  RETURN_IF_UNLIKELY_ERROR;

  TlvDecoder compsD;
  TlvElement_MakeValueDecoder(&nameEle, &compsD);

  n->digestPos.m = NULL;
  n->nComps = 0;
  while (!MbufLoc_IsEnd(&compsD)) {
    TlvElement compEle;
    size_t compLen;
    e = DecodeTlvElement(&compsD, &compEle, &compLen);
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
    size_t len;
    NdnError e = DecodeTlvElement(&d, ele, &len);
    assert(e == NdnError_OK); // cannot error in valid name
  }

  // last DecodeTlvElement invocation was on i-th element
}