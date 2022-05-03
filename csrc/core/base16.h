#ifndef NDNDPDK_CORE_BASE16_H
#define NDNDPDK_CORE_BASE16_H

/** @file */

#include "common.h"

/** @brief Compute base16 buffer size from input of @p size octets. */
#define Base16_BufferSize(size) (2 * (size) + 1)

/**
 * @brief Encode in base16 hexadecimal format.
 * @param[out] output output buffer, must have enough room.
 * @param room output buffer size, too small causes assertion failure.
 * @param input input buffer.
 * @param size input buffer size.
 * @return number of characters written, excluding trailing null character.
 */
__attribute__((nonnull)) static inline int
Base16_Encode(char* output, size_t room, const uint8_t* input, size_t size)
{
  NDNDPDK_ASSERT(room >= Base16_BufferSize(size));
  static char hex[] = "0123456789ABCDEF";
  for (uint16_t i = 0; i < size; ++i) {
    uint8_t b = input[i];
    output[2 * i] = hex[b >> 4];
    output[2 * i + 1] = hex[b & 0x0F];
  }
  output[2 * size] = '\0';
  return 2 * size;
}

#endif // NDNDPDK_CORE_BASE16_H
