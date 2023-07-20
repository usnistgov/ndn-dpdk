#ifndef NDNDPDK_NDNI_TLV_ENCODER_H
#define NDNDPDK_NDNI_TLV_ENCODER_H

/** @file */

#include "common.h"

/** @brief Compute size of VAR-NUMBER. */
static __rte_always_inline uint8_t
TlvEncoder_SizeofVarNum(uint32_t n) {
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
 * @return size of VAR-NUMBER.
 */
__attribute__((nonnull)) static __rte_always_inline size_t
TlvEncoder_WriteVarNum(uint8_t* room, uint32_t n) {
  NDNDPDK_ASSERT(room != NULL);
  if (likely(n < 0xFD)) {
    room[0] = n;
    return 1;
  } else if (likely(n <= UINT16_MAX)) {
    room[0] = 0xFD;
    unaligned_uint16_t* b = RTE_PTR_ADD(room, 1);
    *b = rte_cpu_to_be_16(n);
    return 3;
  } else {
    room[0] = 0xFE;
    unaligned_uint32_t* b = RTE_PTR_ADD(room, 1);
    *b = rte_cpu_to_be_32(n);
    return 5;
  }
}

/**
 * @brief Prepend TLV-TYPE and TLV-LENGTH to mbuf.
 * @param m target mbuf, must have enough headroom.
 */
__attribute__((nonnull)) static __rte_always_inline void
TlvEncoder_PrependTL(struct rte_mbuf* m, uint32_t type, uint32_t length) {
  uint8_t* room = (uint8_t*)rte_pktmbuf_prepend(m, TlvEncoder_SizeofVarNum(type) +
                                                     TlvEncoder_SizeofVarNum(length));
  room += TlvEncoder_WriteVarNum(room, type);
  TlvEncoder_WriteVarNum(room, length);
}

/**
 * @brief Encode constant TLV-TYPE and TLV-LENGTH to unaligned_uint16_t constant.
 * @param type TLV-TYPE number, compile-time constant less than 0xFD.
 * @param length TLV-LENGTH number, compile-time constant no more than 0xFF.
 */
#define TlvEncoder_ConstTL1(type, length)                                                          \
  __extension__({                                                                                  \
    static_assert(__builtin_constant_p((type)), "");                                               \
    static_assert(__builtin_constant_p((length)), "");                                             \
    static_assert((type) < 0xFD, "");                                                              \
    static_assert((length) <= UINT8_MAX, "");                                                      \
    rte_cpu_to_be_16(((type) << 8) | (length));                                                    \
  })

/**
 * @brief Encode constant TLV-TYPE and TLV-LENGTH to unaligned_uint32_t constant.
 * @param type TLV-TYPE number, compile-time constant between 0xFD and 0xFFFF.
 * @param length TLV-LENGTH number, compile-time constant no more than 0xFF.
 */
#define TlvEncoder_ConstTL3(type, length)                                                          \
  __extension__({                                                                                  \
    static_assert(__builtin_constant_p((type)), "");                                               \
    static_assert(__builtin_constant_p((length)), "");                                             \
    static_assert(0xFD <= (type) && (type) <= UINT16_MAX, "");                                     \
    static_assert((length) <= UINT8_MAX, "");                                                      \
    rte_cpu_to_be_32(0xFD000000 | ((type) << 8) | (length));                                       \
  })

#endif // NDNDPDK_NDNI_TLV_ENCODER_H
