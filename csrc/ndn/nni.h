#ifndef NDN_DPDK_NDN_NNI_H
#define NDN_DPDK_NDN_NNI_H

/// \file

#include "common.h"

/** \brief Parse a NonNegativeInteger.
 *  \param[out] n the number.
 */
static __rte_always_inline NdnError
DecodeNni(uint8_t length, const uint8_t* value, uint64_t* n)
{
  switch (length) {
    case 1:
      *n = value[0];
      break;
    case 2:
      *n = rte_be_to_cpu_16(*(unaligned_uint16_t*)(value));
      break;
    case 4:
      *n = rte_be_to_cpu_32(*(unaligned_uint32_t*)(value));
      break;
    case 8:
      *n = rte_be_to_cpu_64(*(unaligned_uint64_t*)(value));
      break;
    default:
      return NdnError_BadNni;
  }
  return NdnError_OK;
}

/** \brief Encode a NonNegativeInteger in minimum size.
 *  \param[out] room output buffer, must have 8 octets
 *  \param n the number
 *  \return actual length
 */
static __rte_always_inline int
EncodeNni(uint8_t* room, uint64_t n)
{
  if (n > UINT16_MAX) {
    if (n > UINT32_MAX) {
      *(unaligned_uint64_t*)room = rte_cpu_to_be_64(n);
      return 8;
    } else {
      *(unaligned_uint32_t*)room = rte_cpu_to_be_32(n);
      return 4;
    }
  } else {
    if (n > UINT8_MAX) {
      *(unaligned_uint16_t*)room = rte_cpu_to_be_16(n);
      return 2;
    } else {
      *room = n;
      return 1;
    }
  }
}

#endif // NDN_DPDK_NDN_NNI_H
