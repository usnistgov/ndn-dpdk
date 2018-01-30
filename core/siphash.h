#ifndef NDN_DPDK_CORE_SIPHASH_H
#define NDN_DPDK_CORE_SIPHASH_H

#include "common.h"
#include "siphash-20121104.h"
#include <rte_memcpy.h>

/// \file

/** \brief a key for SipHash
 */
typedef struct sipkey SipHashKey;

#define SIPHASHKEY_SIZE 16

static inline void
SipHashKey_FromBuffer(SipHashKey* key, const uint8_t buf[SIPHASHKEY_SIZE])
{
  sip_tokey(key, buf);
}

/** \brief context for SipHash
 */
typedef struct siphash SipHash;

/** \brief initialize SipHash-2-4
 */
static inline void
SipHash_Init(SipHash* h, const SipHashKey* key)
{
  sip24_init(h, key);
}

/** \brief write input into SipHash
 */
static inline void
SipHash_Write(SipHash* h, const uint8_t* input, size_t count)
{
  sip24_update(h, input, count);
}

/** \brief finalize SipHash
 *  \return hash value
 */
static inline uint64_t
SipHash_Final(SipHash* h)
{
  return sip24_final(h);
}

/** \brief compute hash value without changing underlying state
 *  \return hash value
 */
static inline uint64_t
SipHash_Sum(const SipHash* h)
{
  SipHash h2;
  rte_memcpy(&h2, h, sizeof(h2));
  h2.p = h2.buf + (h->p - h->buf);
  return sip24_final(&h2);
}

#undef SIP_ROTL
#undef SIP_U32TO8_LE
#undef SIP_U64TO8_LE
#undef SIP_U8TO64_LE
#undef SIPHASH_INITIALIZER
#undef SIP_KEYLEN
#undef sip_keyof
#undef sip_binof
#undef sip_endof
#undef siphash24

#endif // NDN_DPDK_CORE_SIPHASH_H
