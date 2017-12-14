#ifndef NDN_TRAFFIC_DPDK_NDN_NAME_H
#define NDN_TRAFFIC_DPDK_NDN_NAME_H

/// \file

#include "tlv-element.h"

/** \brief maximum number of name components
 */
#define NAME_MAX_INDEXED_COMPS 14

/** \brief TLV Name
 *
 *  This struct stores indices of first \p NAME_MAX_INDEXED_COMPS name components, but permits
 *  arbitrary number of name components.
 */
typedef struct Name
{
  uint16_t nComps;                         ///< number of components
  MbufLoc compPos[NAME_MAX_INDEXED_COMPS]; ///< start position of components
  MbufLoc digestPos; ///< start position of implicit digest component
} Name;
static_assert(sizeof(Name) <= 4 * RTE_CACHE_LINE_SIZE, "");

/** \brief Decode a name.
 *  \param[out] n the name.
 *  \retval NdnError_NameHasComponentAfterDigest unexpected non-digest component after digest.
 */
NdnError DecodeName(TlvDecoder* d, Name* n, size_t* len);

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
    return MbufLoc_Clone(pos, &n->compPos[i]);
  }

  TlvElement ele;
  __Name_GetComp_PastIndexed(n, i, &ele);
  MbufLoc_Clone(pos, &ele.first);
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
  MbufLoc_Clone(&d, &n->compPos[i]);
  size_t len;
  NdnError e = DecodeTlvElement(&d, ele, &len);
  assert(e == NdnError_OK); // cannot error in valid name
}

#endif // NDN_TRAFFIC_DPDK_NDN_NAME_H