#ifndef NDNDPDK_NDNI_NNI_H
#define NDNDPDK_NDNI_NNI_H

/** @file */

#include "common.h"

/**
 * @brief Decode a non-negative integer.
 * @return whether success.
 */
__attribute__((nonnull)) static __rte_always_inline bool
Nni_Decode(uint32_t length, const uint8_t* value, uint64_t* n)
{
  switch (length) {
    case 1:
      *n = value[0];
      return true;
    case 2:
      *n = rte_be_to_cpu_16(*(unaligned_uint16_t*)value);
      return true;
    case 4:
      *n = rte_be_to_cpu_32(*(unaligned_uint32_t*)value);
      return true;
    case 8:
      *n = rte_be_to_cpu_64(*(unaligned_uint64_t*)value);
      return true;
    default:
      *n = 0;
      return false;
  }
}

/**
 * @brief Encode a NonNegativeInteger in minimum size.
 * @param[out] room output buffer, must have 8 octets.
 * @return actual size.
 */
__attribute__((nonnull)) static __rte_always_inline uint8_t
Nni_Encode(uint8_t* room, uint64_t n)
{
  if (n > UINT16_MAX) {
    if (n > UINT32_MAX) {
      unaligned_uint64_t* b = (unaligned_uint64_t*)room;
      *b = rte_cpu_to_be_64(n);
      return 8;
    }

    unaligned_uint32_t* b = (unaligned_uint32_t*)room;
    *b = rte_cpu_to_be_32(n);
    return 4;
  }

  if (n > UINT8_MAX) {
    unaligned_uint16_t* b = (unaligned_uint16_t*)room;
    *b = rte_cpu_to_be_16(n);
    return 2;
  }

  *room = n;
  return 1;
}

#endif // NDNDPDK_NDNI_NNI_H
