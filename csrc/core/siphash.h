#ifndef NDNDPDK_CORE_SIPHASH_H
#define NDNDPDK_CORE_SIPHASH_H

/** @file */

#include "common.h"

#include "../vendor/siphash-20201003.h"

/** @brief A key for SipHash. */
typedef struct sipkey SipHashKey;

#define SIPHASHKEY_SIZE SIP_KEYLEN

__attribute__((nonnull)) static inline void
SipHashKey_FromBuffer(SipHashKey* key, const uint8_t buf[SIPHASHKEY_SIZE])
{
  sip_tokey(key, buf);
}

/** @brief Context for SipHash. */
typedef struct siphash SipHash;

/** @brief Initialize SipHash-2-4 context. */
__attribute__((nonnull)) static inline void
SipHash_Init(SipHash* h, const SipHashKey* key)
{
  sip24_init(h, key);
}

/** @brief Write input into SipHash. */
__attribute__((nonnull)) static inline void
SipHash_Write(SipHash* h, const uint8_t* input, size_t count)
{
  sip24_update(h, input, count);
}

/**
 * @brief Finalize SipHash.
 * @return hash value.
 */
__attribute__((nonnull)) static inline uint64_t
SipHash_Final(SipHash* h)
{
  return sip24_final(h);
}

/**
 * @brief Compute hash value without changing underlying state.
 * @return hash value.
 */
__attribute__((nonnull)) static inline uint64_t
SipHash_Sum(const SipHash* h)
{
  SipHash copy = *h;
  copy.p = RTE_PTR_ADD(copy.buf, RTE_PTR_DIFF(h->p, h->buf));
  return sip24_final(&copy);
}

#undef _SIP_ULL
#undef SIP_ROTL
#undef SIP_U32TO8_LE
#undef SIP_U64TO8_LE
#undef SIP_U8TO64_LE
#undef SIPHASH_INITIALIZER
#undef sip_keyof
#undef sip_endof

#endif // NDNDPDK_CORE_SIPHASH_H
