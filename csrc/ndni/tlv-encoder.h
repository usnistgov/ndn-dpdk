#ifndef NDNDPDK_NDNI_TLV_ENCODER_H
#define NDNDPDK_NDNI_TLV_ENCODER_H

/** @file */

#include "common.h"

/** @brief Compute size of VAR-NUMBER. */
static __rte_always_inline uint8_t
TlvEncoder_SizeofVarNum(uint32_t n)
{
  if (likely(n < 0xFD)) {
    return 1;
  } else if (likely(n <= UINT16_MAX)) {
    return 3;
  } else {
    return 5;
  }
}

/**
 * @brief Write VAR-NUMBER to the given buffer.
 * @param[out] room output buffer, must have sufficient size.
 */
__attribute__((nonnull)) static __rte_always_inline void
TlvEncoder_WriteVarNum(uint8_t* room, uint32_t n)
{
  NDNDPDK_ASSERT(room != NULL);
  if (likely(n < 0xFD)) {
    room[0] = n;
  } else if (likely(n <= UINT16_MAX)) {
    room[0] = 0xFD;
    unaligned_uint16_t* b = RTE_PTR_ADD(room, 1);
    *b = rte_cpu_to_be_16(n);
  } else {
    room[0] = 0xFE;
    unaligned_uint32_t* b = RTE_PTR_ADD(room, 1);
    *b = rte_cpu_to_be_32(n);
  }
}

/**
 * @brief Prepend VAR-NUMBER to mbuf.
 * @param m target mbuf, must have enough headroom.
 */
__attribute__((nonnull)) static __rte_always_inline void
TlvEncoder_PrependVarNum(struct rte_mbuf* m, uint32_t n)
{
  uint8_t* room = (uint8_t*)rte_pktmbuf_prepend(m, TlvEncoder_SizeofVarNum(n));
  TlvEncoder_WriteVarNum(room, n);
}

/**
 * @brief Prepend TLV-TYPE and TLV-LENGTH to mbuf.
 * @param m target mbuf, must have enough headroom.
 */
__attribute__((nonnull)) static __rte_always_inline void
TlvEncoder_PrependTL(struct rte_mbuf* m, uint32_t type, uint32_t length)
{
  uint16_t sizeT = TlvEncoder_SizeofVarNum(type);
  uint16_t sizeL = TlvEncoder_SizeofVarNum(length);
  uint8_t* room = (uint8_t*)rte_pktmbuf_prepend(m, sizeT + sizeL);
  TlvEncoder_WriteVarNum(room, type);
  TlvEncoder_WriteVarNum((uint8_t*)RTE_PTR_ADD(room, sizeT), length);
}

#endif // NDNDPDK_NDNI_TLV_ENCODER_H
