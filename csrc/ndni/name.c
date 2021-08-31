#include "name.h"

#include "../core/siphash.h"
#include "../vendor/pcg_basic.h"

static SipHashKey LName_HashKey_;
uint64_t LName_EmptyHash_;

uint64_t
LName_ComputeHash(LName name)
{
  SipHash h;
  SipHash_Init(&h, &LName_HashKey_);
  SipHash_Write(&h, name.value, name.length);
  return SipHash_Final(&h);
}

RTE_INIT(InitLNameHash)
{
  pcg32_random_t rng;
  // seed with time, because rte_rand() is unavailable before EAL init
  pcg32_srandom_r(&rng, rte_get_tsc_cycles(), 0);

  uint8_t key[SIPHASHKEY_SIZE];
  for (uint8_t* k = key; k != key + SIPHASHKEY_SIZE; ++k) {
    *k = (uint8_t)pcg32_random_r(&rng);
  }
  SipHashKey_FromBuffer(&LName_HashKey_, key);

  LName_EmptyHash_ = LName_ComputeHash((LName){ 0 });
}

int
LName_PrintHex(LName name, char buffer[NameHexBufferLength])
{
  static char hex[] = "0123456789ABCDEF";
  for (uint16_t i = 0; i < name.length; ++i) {
    uint8_t b = name.value[i];
    buffer[2 * i] = hex[b >> 4];
    buffer[2 * i + 1] = hex[b & 0x0F];
  }
  buffer[2 * name.length] = '\0';
  return 2 * name.length;
}

bool
PName_Parse(PName* p, LName l)
{
  *p = (const PName){ .value = l.value, .length = l.length, .firstNonGeneric = -1 };

  uint16_t pos = 0, end = 0, type = 0, length = 0;
  while (likely(LName_Component(PName_ToLName(p), &pos, &type, &length))) {
    end = (pos += length);
    if (likely(p->nComps < PNameCachedComponents)) {
      p->comp_[p->nComps] = pos;
    }
    if (unlikely(type != TtGenericNameComponent && p->firstNonGeneric < 0)) {
      p->firstNonGeneric = p->nComps;
    }
    ++p->nComps;
  }
  if (unlikely(end != pos)) { // truncated component
    return false;
  }

  if (unlikely(type == TtImplicitSha256DigestComponent) && likely(length == ImplicitDigestLength)) {
    p->hasDigestComp = true;
  }
  return true;
}

void
PName_PrepareHashes_(PName* p)
{
  SipHash h;
  SipHash_Init(&h, &LName_HashKey_);

  uint16_t pos = 0;
  for (uint16_t i = 0, last = RTE_MIN(p->nComps, PNameCachedComponents); i < last; ++i) {
    SipHash_Write(&h, (const uint8_t*)RTE_PTR_ADD(p->value, pos), p->comp_[i] - pos);
    p->hash_[i] = SipHash_Sum(&h);
    pos = p->comp_[i];
  }

  p->hasHashes_ = true;
}
