#include "name.h"
#include "tlv-varnum.h"

#include "../core/pcg_basic.h"
#include "../core/siphash.h"

static SipHashKey theNameHashKey;
uint64_t NameHash_Empty_;

RTE_INIT(NameHash_Init)
{
  pcg32_random_t rng;
  // seed with time, because rte_rand() is unavailable before EAL init
  pcg32_srandom_r(&rng, rte_get_tsc_cycles(), 0);

  uint8_t key[SIPHASHKEY_SIZE];
  for (uint8_t* k = key; k != key + SIPHASHKEY_SIZE; ++k) {
    *k = (uint8_t)pcg32_random_r(&rng);
  }
  SipHashKey_FromBuffer(&theNameHashKey, key);

  SipHash h;
  SipHash_Init(&h, &theNameHashKey);
  NameHash_Empty_ = SipHash_Final(&h);
}

uint64_t
LName_ComputeHash_(uint16_t length, const uint8_t* value)
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

int
LName_ToString(LName n, char* buf, size_t bufsz)
{
  int count = 0;
  for (uint16_t i = 0; i < n.length; ++i) {
    int res = snprintf(buf, bufsz, "%02X", n.value[i]);
    if (unlikely(res != 2 || bufsz <= 2)) {
      *buf = '\0';
      break;
    }
    count += res;
    buf = RTE_PTR_ADD(buf, res);
    bufsz -= res;
  }
  return count;
}

static bool
IsValidNameComponentType(uint32_t type)
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
    uint32_t compT;
    int sizeofT = DecodeVarNum(value + off, length - off, &compT);
    if (unlikely(sizeofT <= 0)) {
      return -sizeofT;
    }

    uint32_t compL;
    int sizeofL =
      DecodeVarNum(value + off + sizeofT, length - off - sizeofT, &compL);
    if (unlikely(sizeofL <= 0)) {
      return -sizeofL;
    }

    off += sizeofT + sizeofL + compL;
    if (unlikely(off > length)) {
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
      n->comp[n->nComps] = off;
    }
    ++n->nComps;
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
PName_SeekCompEnd_(const PName* n, const uint8_t* input, uint16_t i)
{
  assert(i >= PNAME_N_CACHED_COMPS);
  uint16_t off = n->comp[PNAME_N_CACHED_COMPS - 1];
  for (uint16_t j = PNAME_N_CACHED_COMPS - 1; j < i; ++j) {
    uint32_t compT, compL;
    off += DecodeVarNum(input + off, n->nOctets - off, &compT);
    off += DecodeVarNum(input + off, n->nOctets - off, &compL);
    off += compL;
  }
  return off;
}

void
PName_HashToCache_(PName* n, const uint8_t* input)
{
  SipHash h;
  SipHash_Init(&h, &theNameHashKey);

  uint16_t off = 0;
  for (uint16_t i = 0, last = RTE_MIN(n->nComps, PNAME_N_CACHED_COMPS);
       i < last;
       ++i) {
    SipHash_Write(&h, input + off, n->comp[i] - off);
    n->hash[i] = SipHash_Sum(&h);
    off = n->comp[i];
  }

  n->hasHashes = true;
}
