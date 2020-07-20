#ifndef NDNDPDK_NDNI_TLV_DECODER_H
#define NDNDPDK_NDNI_TLV_DECODER_H

/** @file */

#include "nni.h"

/** @brief Determine if a TLV-TYPE is critical for evolvability purpose. */
static __rte_always_inline bool
TlvDecoder_IsCriticalType(uint32_t type)
{
  return type <= 0x1F || (type % 2) == 1;
}

/** @brief TLV decoder. */
typedef struct TlvDecoder
{
  struct rte_mbuf* p; ///< first segment
  struct rte_mbuf* m; ///< current segment
  uint32_t length;    ///< remaining byte length
  uint16_t offset;    ///< offset within current segment
} TlvDecoder;

/** @brief Create a TlvDecoder. */
__attribute__((nonnull)) static inline void
TlvDecoder_Init(TlvDecoder* d, struct rte_mbuf* p)
{
  d->p = p;
  d->m = p;
  d->offset = 0;
  d->length = p->pkt_len;
}

/**
 * @brief Skip @p count octets.
 * @pre Decoder has no less than @p count remaining octets.
 */
__attribute__((nonnull)) static inline void
TlvDecoder_Skip(TlvDecoder* d, uint32_t count)
{
  NDNDPDK_ASSERT(count <= d->length);
  for (uint32_t remain = count; remain > 0;) {
    uint32_t here = d->m->data_len - d->offset;
    if (likely(remain < here)) {
      d->offset += remain;
      break;
    }
    remain -= here;
    d->m = d->m->next;
    d->offset = 0;
  }
  d->length -= count;
}

__attribute__((nonnull, returns_nonnull)) static inline const uint8_t*
TlvDecoder_Read_Contiguous_(TlvDecoder* d, uint16_t count)
{
  uint16_t here = d->m->data_len - d->offset;
  NDNDPDK_ASSERT(count <= here);
  const uint8_t* output = rte_pktmbuf_mtod_offset(d->m, const uint8_t*, d->offset);

  d->length -= count;
  if (likely(count < here)) {
    d->offset += count;
  } else {
    d->m = d->m->next;
    d->offset = 0;
  }

  return output;
}

__attribute__((nonnull)) void
TlvDecoder_Read_NonContiguous_(TlvDecoder* d, uint8_t* output, uint16_t count);

/**
 * @brief Read next @p count octets in contiguous memory.
 * @param scratch a buffer for copying from non-contiguous memory; must have at least @c count room.
 * @return pointer to current position in linear memory.
 * @pre Decoder has no less than @p count remaining octets.
 * @post Decoder is advanced by @c count octets.
 */
__attribute__((nonnull, returns_nonnull)) static inline const uint8_t*
TlvDecoder_Read(TlvDecoder* d, uint8_t* scratch, uint16_t count)
{
  NDNDPDK_ASSERT(count <= d->length);
  if (unlikely(count == 0)) {
    return scratch;
  }
  if (likely(count <= d->m->data_len - d->offset)) {
    return TlvDecoder_Read_Contiguous_(d, count);
  }
  TlvDecoder_Read_NonContiguous_(d, scratch, count);
  return scratch;
}

/**
 * @brief Clone next @p count octets to indirect mbufs.
 * @param[out] lastseg if non-NULL, receive the pointer to the last segment.
 * @return indirect mbufs.
 * @retval NULL allocation failure.
 * @pre Decoder has no less than @p count remaining octets.
 * @post Decoder is advanced by @c count octets.
 */
__attribute__((nonnull(1, 3))) struct rte_mbuf*
TlvDecoder_Clone(TlvDecoder* d, uint32_t count, struct rte_mempool* indirectMp,
                 struct rte_mbuf** lastseg);

__attribute__((nonnull)) const uint8_t*
TlvDecoder_Linearize_NonContiguous_(TlvDecoder* d, uint16_t count);

/**
 * @brief Copy next @p count octets to contiguous memory in allocated mbuf.
 * @return pointer to current position in contiguous memory.
 * @retval NULL allocation failure in @c d->m->pool .
 * @pre @c d->m is a uniquely owned direct mbuf.
 * @pre Decoder has no less than @p count remaining octets.
 * @post Decoder is advanced by @c count octets.
 */
__attribute__((nonnull)) static inline const uint8_t*
TlvDecoder_Linearize(TlvDecoder* d, uint16_t count)
{
  NDNDPDK_ASSERT(count <= d->length);
  if (unlikely(count == 0)) {
    return NULL;
  }
  if (likely(count < d->m->data_len - d->offset)) {
    return TlvDecoder_Read_Contiguous_(d, count);
  }
  return TlvDecoder_Linearize_NonContiguous_(d, count);
}

/**
 * @brief Read a VAR-NUMBER.
 * @return whether success.
 * @post Decoder is advanced after the VAR-NUMBER.
 */
__attribute__((nonnull)) static inline bool
TlvDecoder_ReadVarNum(TlvDecoder* d, uint32_t* n)
{
  if (unlikely(d->length < 1)) {
    return false;
  }
  uint8_t scratch[4];
  uint8_t first = *TlvDecoder_Read(d, scratch, 1);
  switch (first) {
    case 0xFD:
      if (likely(d->length >= 2)) {
        const unaligned_uint16_t* b = (const unaligned_uint16_t*)TlvDecoder_Read(d, scratch, 2);
        *n = rte_be_to_cpu_16(*b);
        return likely(*n >= 0xFD);
      }
      return false;
    case 0xFE:
      if (likely(d->length >= 4)) {
        const unaligned_uint32_t* b = (const unaligned_uint32_t*)TlvDecoder_Read(d, scratch, 4);
        *n = rte_be_to_cpu_32(*b);
        return likely(*n > UINT16_MAX);
      }
      return false;
    case 0xFF:
      return false;
    default:
      *n = first;
      return true;
  }
}

/**
 * @brief Read TLV-TYPE and TLV-LENGTH.
 * @param[out] length TLV-LENGTH.
 * @return TLV-TYPE number.
 * @retval 0 truncated packet.
 * @post Decoder is advanced after TLV-LENGTH.
 */
__attribute__((nonnull)) static inline uint32_t
TlvDecoder_ReadTL(TlvDecoder* d, uint32_t* length)
{
  *length = 0;
  uint32_t type;
  if (likely(TlvDecoder_ReadVarNum(d, &type)) && likely(TlvDecoder_ReadVarNum(d, length)) &&
      likely(d->length >= *length)) {
    return type;
  }
  return 0;
}

/**
 * @brief Iterate over TLV elements.
 * @code
 * TlvDecoder_EachTL (&decoder, type, length) {
 *   // type is the TLV-TYPE
 *   // length is the TLV-LENGTH
 *   TlvDecoder_Skip(&decoder, length); // must advance over the length
 * }
 * @endcode
 */
#define TlvDecoder_EachTL(d, typeVar, lengthVar)                                                   \
  for (uint32_t lengthVar, typeVar = TlvDecoder_ReadTL((d), &lengthVar); typeVar != 0;             \
       typeVar = TlvDecoder_ReadTL((d), &lengthVar))

/**
 * @brief Read TLV-VALUE by creating a sub decoder.
 * @param d parent decoder.
 * @param length TLV-LENGTH of current element.
 * @param vd value decoder.
 * @return whether success.
 * @post Parent decoder is advanced after the TLV-VALUE.
 */
__attribute__((nonnull)) static inline void
TlvDecoder_MakeValueDecoder(TlvDecoder* restrict d, uint32_t length, TlvDecoder* restrict vd)
{
  *vd = *d;
  vd->length = length;
  TlvDecoder_Skip(d, length);
}

/**
 * @brief Read non-negative integer.
 * @param max inclusive maximum acceptable value.
 * @return whether success.
 * @post Decoder is advanced after the number.
 */
__attribute__((nonnull)) static __rte_always_inline bool
TlvDecoder_ReadNni(TlvDecoder* d, uint32_t length, uint64_t max, uint64_t* n)
{
  uint8_t scratch[8];
  const uint8_t* value = TlvDecoder_Read(d, scratch, RTE_MIN(sizeof(scratch), length));
  return likely(Nni_Decode(length, value, n) && *n <= max);
}

#define TlvDecoder_ReadNniTo4_(d, length, max, ptr)                                                \
  __extension__({                                                                                  \
    uint64_t tmax = 0;                                                                             \
    switch (sizeof(*(ptr))) {                                                                      \
      case sizeof(uint8_t):                                                                        \
        tmax = UINT8_MAX;                                                                          \
        break;                                                                                     \
      case sizeof(uint16_t):                                                                       \
        tmax = UINT16_MAX;                                                                         \
        break;                                                                                     \
      case sizeof(uint32_t):                                                                       \
        tmax = UINT32_MAX;                                                                         \
        break;                                                                                     \
      case sizeof(uint64_t):                                                                       \
        tmax = UINT64_MAX;                                                                         \
        break;                                                                                     \
    }                                                                                              \
    uint64_t value;                                                                                \
    bool ok = TlvDecoder_ReadNni((d), (length), RTE_MIN((max), tmax), &value);                     \
    *(ptr) = value;                                                                                \
    ok;                                                                                            \
  })
#define TlvDecoder_ReadNniTo3_(d, length, ptr)                                                     \
  TlvDecoder_ReadNniTo4_((d), (length), UINT64_MAX, (ptr))
#define TlvDecoder_ReadNniToArg5_(a1, a2, a3, a4, a5, ...) a5
#define TlvDecoder_ReadNniToChoose_(...)                                                           \
  TlvDecoder_ReadNniToArg5_(__VA_ARGS__, TlvDecoder_ReadNniTo4_, TlvDecoder_ReadNniTo3_, )

/**
 * @brief Read non-negative integer to a pointer of any unsigned type.
 * @code
 * bool ok = TlvDecoder_ReadNniTo(&decoder, length, max, &var);
 * bool ok = TlvDecoder_ReadNniTo(&decoder, length, &var);
 * // Target variable can be uint8_t, uint16_t, uint32_t, or uint64_t.
 * // max defaults to, and is reduced to the maximum value assignable to the target variable.
 * @endcode
 * @return whether success.
 * @post Decoder is advanced after the number.
 */
#define TlvDecoder_ReadNniTo(...) (TlvDecoder_ReadNniToChoose_(__VA_ARGS__)(__VA_ARGS__))

#endif // NDNDPDK_NDNI_TLV_DECODER_H
