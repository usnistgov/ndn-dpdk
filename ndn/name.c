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
  n->hasDigestComp = false;
  n->hasPrefixHashes = false;

  TlvDecoder compsD;
  TlvElement_MakeValueDecoder(&nameEle, &compsD);

  while (!MbufLoc_IsEnd(&compsD)) {
    if (unlikely(n->hasDigestComp)) {
      return NdnError_NameHasComponentAfterDigest;
    }

    TlvElement compEle;
    e = DecodeTlvElement(&compsD, &compEle);
    RETURN_IF_UNLIKELY_ERROR;

    if (unlikely(compEle.type == TT_ImplicitSha256DigestComponent)) {
      if (unlikely(compEle.length != 32)) {
        return NdnError_BadDigestComponentLength;
      }
      n->hasDigestComp = true;
    }

    if (likely(n->nComps < NAME_MAX_INDEXED_COMPS)) {
      MbufLoc_Copy(&n->comps[n->nComps].pos, &compEle.first);
    }
    ++n->nComps;
  }

  return NdnError_OK;
}

const uint8_t*
Name_LinearizeComps(const Name* n, uint8_t scratch[NAME_MAX_LENGTH])
{
  assert(n->nOctets <= NAME_MAX_LENGTH);

  MbufLoc ml;
  MbufLoc_Copy(&ml, &n->comps[0].pos);

  uint32_t nRead;
  const uint8_t* linear = MbufLoc_Read(&ml, scratch, n->nOctets, &nRead);
  assert(nRead == n->nOctets);
  return linear;
}

void
__Name_GetComp_PastIndexed(const Name* n, uint16_t i, TlvElement* ele)
{
  assert(n->nComps >= NAME_MAX_INDEXED_COMPS);
  assert(i >= NAME_MAX_INDEXED_COMPS);

  TlvDecoder d;
  uint16_t j = NAME_MAX_INDEXED_COMPS - 1;
  MbufLoc_Copy(&d, &n->comps[j].pos);
  for (; j <= i; ++j) {
    NdnError e = DecodeTlvElement(&d, ele);
    assert(e == NdnError_OK); // cannot error in valid name
  }

  // last DecodeTlvElement invocation was on i-th element
}

static void
__Name_ComputePrefixHashes_WriteToSipHash(void* h, const struct rte_mbuf* m,
                                          uint16_t off, uint16_t len)
{
  SipHash_Write((SipHash*)h, rte_pktmbuf_mtod_offset(m, const uint8_t*, off),
                len);
}

void
__Name_ComputePrefixHashes(Name* n)
{
  SipHash h;
  SipHash_Init(&h, &theNameHashKey);

  for (uint16_t i = 0; i < n->nComps && i < NAME_MAX_INDEXED_COMPS; ++i) {
    TlvElement ele;
    Name_GetComp(n, i, &ele);

    __MbufLoc_AdvanceWithCb(&ele.first, ele.size,
                            __Name_ComputePrefixHashes_WriteToSipHash, &h);
    n->comps[i].hash = SipHash_Sum(&h);
  }

  n->hasPrefixHashes = true;
}

uint64_t
__Name_ComputePrefixHash_PastIndexed(const Name* n, uint16_t i)
{
  MbufLoc begin;
  Name_GetCompPos(n, 0, &begin);
  TlvElement end;
  Name_GetComp(n, i, &end);
  ptrdiff_t size = MbufLoc_Diff(&begin, &end.last);

  SipHash h;
  SipHash_Init(&h, &theNameHashKey);
  __MbufLoc_AdvanceWithCb(&begin, size,
                          __Name_ComputePrefixHashes_WriteToSipHash, &h);
  return SipHash_Final(&h);
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

  uint8_t scratchL[NAME_MAX_LENGTH];
  uint8_t scratchR[NAME_MAX_LENGTH];
  const uint8_t* compsL = Name_LinearizeComps(lhs, scratchL);
  const uint8_t* compsR = Name_LinearizeComps(rhs, scratchR);

  uint16_t minOctets =
    lhs->nOctets <= rhs->nOctets ? lhs->nOctets : rhs->nOctets;
  int cmp = memcmp(compsL, compsR, minOctets);
  if (cmp != 0) {
    return ((cmp > 0) - (cmp < 0)) << 1;
  }
  cmp = lhs->nComps - rhs->nComps;
  return (cmp > 0) - (cmp < 0);
}
