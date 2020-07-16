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

RTE_INIT(LName_HashInit_)
{
  pcg32_random_t rng;
  // seed with time, because rte_rand() is unavailable before EAL init
  pcg32_srandom_r(&rng, rte_get_tsc_cycles(), 0);

  uint8_t key[SIPHASHKEY_SIZE];
  for (uint8_t* k = key; k != key + SIPHASHKEY_SIZE; ++k) {
    *k = (uint8_t)pcg32_random_r(&rng);
  }
  SipHashKey_FromBuffer(&LName_HashKey_, key);

  LName_EmptyHash_ = LName_ComputeHash(LName_Empty());
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

static __rte_always_inline bool
PName_ParseVarNum_(const PName* p, uint16_t* pos, uint16_t* n, bool allowZero)
{
  if (unlikely(*pos >= p->length)) {
    return false;
  }

  switch (p->value[*pos]) {
    case 0x00:
      *n = 0;
      *pos += 1;
      return allowZero;
    case 0xFD: {
      if (unlikely(*pos + 2 >= p->length)) {
        return false;
      }
      *n = rte_be_to_cpu_16(*(unaligned_uint16_t*)(&p->value[(*pos) + 1]));
      if (unlikely(*n < 0xFD)) {
        return false;
      }
      *pos += 3;
      return true;
    }
    case 0xFE:
    case 0xFF:
      return false;
    default:
      *n = p->value[*pos];
      *pos += 1;
      return true;
  }
}

static __rte_always_inline bool
PName_ParseComponent_(const PName* p, uint16_t* pos, uint16_t* type, uint16_t* length)
{
  return likely(PName_ParseVarNum_(p, pos, type, false)) &&
         likely(PName_ParseVarNum_(p, pos, length, true)) && likely((*pos += *length) <= p->length);
}

bool
PName_Parse(PName* p, LName l)
{
  *p = (const PName){ 0 };
  p->value = l.value;
  p->length = l.length;

  uint16_t pos = 0, end = 0, type = 0, length = 0;
  while (likely(PName_ParseComponent_(p, &pos, &type, &length))) {
    if (likely(p->nComps < PNameCachedComponents)) {
      p->comp_[p->nComps] = pos;
    }
    end = pos;
    ++p->nComps;
  }
  if (unlikely(end != pos)) { // truncated component
    return false;
  }

  p->hasDigestComp =
    unlikely(type == TtImplicitSha256DigestComponent) && length == ImplicitDigestLength;
  return true;
}

LName
PName_GetPrefix_Uncached_(const PName* p, int n)
{
  if (unlikely(n == (int)p->nComps)) {
    return PName_ToLName(p);
  }

  uint16_t pos = 0;
  for (int i = 0; i < n; ++i) {
    uint16_t type, length;
    PName_ParseComponent_(p, &pos, &type, &length);
  }
  return LName_Init(pos, p->value);
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
