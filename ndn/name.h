#ifndef NDN_DPDK_NDN_NAME_H
#define NDN_DPDK_NDN_NAME_H

/// \file

#include "namehash.h"
#include "tlv-element.h"

/** \brief maximum supported name length (TLV-LENGTH of Name element)
 */
#define NAME_MAX_LENGTH 2048

/** \brief Indicate the result of name comparison.
 */
typedef enum NameCompareResult {
  NAMECMP_LT = -2,      ///< \p lhs is less than, but not a prefix of \p rhs
  NAMECMP_LPREFIX = -1, ///< \p lhs is a prefix of \p rhs
  NAMECMP_EQUAL = 0,    ///< \p lhs and \p rhs are equal
  NAMECMP_RPREFIX = 1,  ///< \p rhs is a prefix of \p lhs
  NAMECMP_GT = 2        ///< \p rhs is less than, but not a prefix of \p lhs
} NameCompareResult;

/** \brief Name in linear buffer.
 */
typedef struct LName
{
  const uint8_t* value;
  uint16_t length;
} LName;

uint64_t LName_ComputeHash(LName n);

/** \brief Compare two names in linear buffers.
 */
NameCompareResult LName_Compare(LName lhs, LName rhs);

#endif // NDN_DPDK_NDN_NAME_H