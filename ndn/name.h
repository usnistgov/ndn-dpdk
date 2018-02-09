#ifndef NDN_DPDK_NDN_NAME_H
#define NDN_DPDK_NDN_NAME_H

/// \file

#include "namehash.h"
#include "tlv-element.h"

/** \brief Maximum supported name length (TLV-LENGTH of Name element).
 */
#define NAME_MAX_LENGTH 2048

/** \brief Number of name components whose information are cached in Name struct
 *         for efficient processing.
 */
#define NAME_N_CACHED_COMPS 18

/** \brief Parsed Name element.
 */
typedef struct PName
{
  uint16_t nOctets;   ///< TLV-LENGTH of Name element
  uint16_t nComps;    ///< number of components
  bool hasDigestComp; ///< ends with digest component?

  bool hasHashes; ///< (pvt) are hash[i] computed?
  uint16_t
    comp[NAME_N_CACHED_COMPS]; ///< (pvt) start offset of i+1-th component
  uint64_t hash[NAME_N_CACHED_COMPS]; ///< (pvt) hash of i+1-component prefix
} PName;
static_assert(sizeof(PName) <= 3 * RTE_CACHE_LINE_SIZE, "");

/** \brief Parse a name from memory buffer.
 *  \param length TLV-LENGTH of Name element
 *  \param value TLV-VALUE of Name element
 *  \retval NdnError_OK success
 *  \retval NdnError_NameTooLong TLV-LENGTH exceeds \p NAME_MAX_LENGTH
 *  \retval NdnError_BadNameComponentType component type not in 1-32767 range
 *  \retval NdnError_BadDigestComponentLength ImplicitSha256DigestComponent is not 32 octets
 *  \retval NdnError_NameHasComponentAfterDigest ImplicitSha256DigestComponent is not at last
 */
NdnError PName_Parse(PName* n, uint32_t length, const uint8_t* value);

/** \brief Parse a name from TlvElement.
 *  \param ele TLV Name element, TLV-TYPE must be TT_Name
 *  \retval NdnError_Fragmented TLV-VALUE is not in consecutive memory
 *  \return return value of \p PName_Parse
 */
NdnError PName_FromElement(PName* n, const TlvElement* ele);

uint16_t __PName_SeekCompStart(PName* n, const uint8_t* input, uint16_t i);

/** \brief Get start offset of i-th component.
 *  \param input a buffer containing TLV-VALUE of Name element
 *  \param i component index, must be less than n->nComps
 */
static uint16_t
PName_GetCompStart(PName* n, const uint8_t* input, uint16_t i)
{
  assert(i < n->nComps);
  if (i == 0) {
    return 0;
  }
  if (likely(i <= NAME_N_CACHED_COMPS)) {
    return n->comp[i - 1];
  }
  return __PName_SeekCompStart(n, input, i);
}

/** \brief Get past-end offset of i-th component.
 *  \param input a buffer containing TLV-VALUE of Name element
 *  \param i component index, must be less than n->nComps
 */
static uint16_t
PName_GetCompEnd(PName* n, const uint8_t* input, uint16_t i)
{
  if (i == n->nComps - 1) {
    return n->nOctets;
  }
  return PName_GetCompStart(n, input, i + 1);
}

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