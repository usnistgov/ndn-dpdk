#ifndef NDN_DPDK_NDN_TLV_DECODE_H
#define NDN_DPDK_NDN_TLV_DECODE_H

/// \file

#include "common.h"

static uint32_t
__ParseVarNum16(const uint8_t* input, uint32_t rem, uint64_t* n)
{
  if (unlikely(rem < 3)) {
    return 0;
  }
  *n = rte_be_to_cpu_16(*(unaligned_uint16_t*)(input + 1));
  return 3;
}

static uint32_t
__ParseVarNum32(const uint8_t* input, uint32_t rem, uint64_t* n)
{
  if (unlikely(rem < 5)) {
    return 0;
  }
  *n = rte_be_to_cpu_32(*(unaligned_uint32_t*)(input + 1));
  return 5;
}

static uint32_t
__ParseVarNum64(const uint8_t* input, uint32_t rem, uint64_t* n)
{
  if (unlikely(rem < 9)) {
    return 0;
  }
  *n = rte_be_to_cpu_64(*(unaligned_uint64_t*)(input + 1));
  return 9;
}

typedef uint32_t (*__ParseVarNumSized)(const uint8_t* input, uint32_t rem,
                                       uint64_t* n);

static const __ParseVarNumSized __ParseVarNum_Jmp[3] = {
  __ParseVarNum16, __ParseVarNum32, __ParseVarNum64,
};

/** \brief Parse a TLV-TYPE or TLV-LENGTH number.
 *  \param[out] n the number.
 *  \return number of consumed bytes, or 0 if input is incomplete.
 */
static __rte_always_inline uint32_t
ParseVarNum(const uint8_t* input, uint32_t rem, uint64_t* n)
{
  if (unlikely(rem == 0)) {
    return 0;
  }

  uint8_t firstOctet = *input;
  int jmpIndex = firstOctet - 253;
  if (likely(jmpIndex < 0)) {
    *n = firstOctet;
    return 1;
  }

  return __ParseVarNum_Jmp[jmpIndex](input, rem, n);
}

/** \brief Parse TLV-TYPE and TLV-LENGTH.
 *  \param[out] type TLV-TYPE number.
 *  \param[out] length TLV-LENGTH number.
 *  \return number of consumed bytes, or 0 if input is incomplete.
 */
static __rte_always_inline uint32_t
ParseTlvTypeLength(const uint8_t* input, uint32_t rem, uint64_t* type,
                   uint64_t* length)
{
  uint32_t sizeofType = ParseVarNum(input, rem, type);
  uint32_t sizeofLength =
    ParseVarNum(input + sizeofType, rem - sizeofLength, length);
  return sizeofType + sizeofLength;
}

#endif // NDN_DPDK_NDN_TLV_DECODE_POS_H
