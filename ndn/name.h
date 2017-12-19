#ifndef NDN_DPDK_NDN_NAME_H
#define NDN_DPDK_NDN_NAME_H

/// \file

#include "tlv-element.h"

/** \brief maximum supported name length (TLV-LENGTH of Name element)
 */
#define NAME_MAX_LENGTH 2048

/** \brief maximum number of name components for efficient processing
 */
#define NAME_MAX_INDEXED_COMPS 14

/** \brief TLV Name
 *
 *  This struct stores indices of first \p NAME_MAX_INDEXED_COMPS name components, but permits
 *  arbitrary number of name components.
 */
typedef struct Name
{
  uint16_t nOctets;                        ///< TLV-LENGTH of Name element
  uint16_t nComps;                         ///< number of components
  MbufLoc compPos[NAME_MAX_INDEXED_COMPS]; ///< start position of components
  MbufLoc digestPos; ///< start position of implicit digest component
} Name;
static_assert(sizeof(Name) <= 4 * RTE_CACHE_LINE_SIZE, "");

/** \brief Decode a name.
 *  \param[out] n the name.
 *  \retval NdnError_NameHasComponentAfterDigest unexpected non-digest component after digest.
 */
NdnError DecodeName(TlvDecoder* d, Name* n);

/** \brief Test whether a name contains an implicit digest component.
 */
static inline bool
Name_HasDigest(const Name* n)
{
  return !MbufLoc_IsEnd(&n->digestPos);
}

void __Name_GetComp_PastIndexed(const Name* n, uint16_t i, TlvElement* ele);

/** \brief Get position of i-th name component.
 *  \param i name component index; <tt>0 <= i < nComps</tt>.
 *  \param[out] pos start position of i-th name component.
 */
static inline void
Name_GetCompPos(const Name* n, uint16_t i, MbufLoc* pos)
{
  assert(i < n->nComps);

  if (likely(i < NAME_MAX_INDEXED_COMPS)) {
    return MbufLoc_Copy(pos, &n->compPos[i]);
  }

  TlvElement ele;
  __Name_GetComp_PastIndexed(n, i, &ele);
  MbufLoc_Copy(pos, &ele.first);
}

/** \brief Parse i-th name component.
 *  \param i name component index; <tt>0 <= i < nComps</tt>.
 *  \param[out] ele the element.
 */
static inline void
Name_GetComp(const Name* n, uint16_t i, TlvElement* ele)
{
  assert(i < n->nComps);

  if (unlikely(i >= NAME_MAX_INDEXED_COMPS)) {
    return __Name_GetComp_PastIndexed(n, i, ele);
  }

  TlvDecoder d;
  MbufLoc_Copy(&d, &n->compPos[i]);
  NdnError e = DecodeTlvElement(&d, ele);
  assert(e == NdnError_OK); // cannot error in valid name
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

/** \brief Compare two names.
 */
NameCompareResult Name_Compare(const Name* lhs, const Name* rhs);

#endif // NDN_DPDK_NDN_NAME_H