#include "encode-interest.h"
#include "tlv-encoder.h"

#include <rte_random.h>

void
NonceGen_Init(NonceGen* g)
{
  pcg32_srandom_r(&g->rng, rte_rand(), rte_rand());
}

uint16_t
__InterestTemplate_Prepare(InterestTemplate* tpl,
                           uint8_t* buffer,
                           uint16_t bufferSize,
                           const uint8_t* fhV)
{
  tpl->bufferOff = 0;
  uint16_t size = 0;
  if (tpl->canBePrefix) {
    size += SizeofVarNum(TT_CanBePrefix) + SizeofVarNum(0);
  }
  if (tpl->mustBeFresh) {
    size += SizeofVarNum(TT_MustBeFresh) + SizeofVarNum(0);
  }
  if (tpl->fhL > 0) {
    size += SizeofVarNum(TT_ForwardingHint) + SizeofVarNum(tpl->fhL) + tpl->fhL;
  }
  {
    size += SizeofVarNum(TT_Nonce) + SizeofVarNum(4);
    tpl->nonceOff = size;
    while (tpl->nonceOff % 4 != 0) {
      ++tpl->bufferOff;
      ++tpl->nonceOff;
    }
    size += 4;
  }
  if (tpl->lifetime != DEFAULT_INTEREST_LIFETIME) {
    size += SizeofVarNum(TT_InterestLifetime) + SizeofVarNum(4) + 4;
  }
  {
    size += SizeofVarNum(TT_HopLimit) + SizeofVarNum(1) + 1;
  }
  if (size > bufferSize) {
    return tpl->bufferOff + size;
  }

  uint8_t* p = buffer + tpl->bufferOff;
  if (tpl->canBePrefix) {
    p = EncodeVarNum(p, TT_CanBePrefix);
    p = EncodeVarNum(p, 0);
  }
  if (tpl->mustBeFresh) {
    p = EncodeVarNum(p, TT_MustBeFresh);
    p = EncodeVarNum(p, 0);
  }
  if (tpl->fhL > 0) {
    p = EncodeVarNum(p, TT_ForwardingHint);
    p = EncodeVarNum(p, tpl->fhL);
    rte_memcpy(p, fhV, tpl->fhL);
    p += tpl->fhL;
  }
  {
    p = EncodeVarNum(p, TT_Nonce);
    p = EncodeVarNum(p, 4);
    assert(p == buffer + tpl->nonceOff);
    p += 4;
  }
  if (tpl->lifetime != DEFAULT_INTEREST_LIFETIME) {
    p = EncodeVarNum(p, TT_InterestLifetime);
    p = EncodeVarNum(p, 4);
    rte_be32_t lifetimeV = rte_cpu_to_be_32(tpl->lifetime);
    rte_memcpy(p, &lifetimeV, 4);
    p += 4;
  }
  {
    p = EncodeVarNum(p, TT_HopLimit);
    p = EncodeVarNum(p, 1);
    *p++ = (uint8_t)tpl->hopLimit;
  }
  assert(p == buffer + tpl->bufferOff + size);
  tpl->bufferSize = size;
  return 0;
}

void
__EncodeInterest(struct rte_mbuf* m,
                 const InterestTemplate* tpl,
                 uint8_t* preparedBuffer,
                 uint16_t nameSuffixL,
                 const uint8_t* nameSuffixV,
                 uint32_t nonce,
                 uint16_t paramL,
                 const uint8_t* paramV,
                 const uint8_t* namePrefixV)
{
  assert(rte_pktmbuf_headroom(m) >= EncodeInterest_GetHeadroom());
  assert(rte_pktmbuf_tailroom(m) >=
         EncodeInterest_GetTailroom(tpl, nameSuffixL, paramL));
  TlvEncoder* en = MakeTlvEncoder(m);

  AppendVarNum(en, TT_Name);
  AppendVarNum(en, tpl->namePrefix.length + nameSuffixL);
  if (likely(tpl->namePrefix.length > 0)) {
    rte_memcpy(rte_pktmbuf_append(m, tpl->namePrefix.length),
               namePrefixV,
               tpl->namePrefix.length);
  }
  if (likely(nameSuffixL > 0)) {
    rte_memcpy(rte_pktmbuf_append(m, nameSuffixL), nameSuffixV, nameSuffixL);
  }

  *(uint32_t*)(preparedBuffer + tpl->nonceOff) = nonce;
  rte_memcpy(rte_pktmbuf_append(m, tpl->bufferSize),
             preparedBuffer + tpl->bufferOff,
             tpl->bufferSize);

  if (paramL > 0) {
    AppendVarNum(en, TT_ApplicationParameters);
    AppendVarNum(en, paramL);
    rte_memcpy(rte_pktmbuf_append(m, paramL), paramV, paramL);
  }

  PrependVarNum(en, m->pkt_len);
  PrependVarNum(en, TT_Interest);
}
