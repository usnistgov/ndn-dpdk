#ifndef NDN_DPDK_NDN_NAME1_H
#define NDN_DPDK_NDN_NAME1_H

/// \file

#include "name.h"

/** \brief maximum number of name components for efficient processing
 */
#define NAME_MAX_INDEXED_COMPS 14

/** \brief TLV Name component record
 */
typedef struct NameCompRecord
{
  MbufLoc pos;   ///< start position of component's TLV-TYPE
  uint64_t hash; ///< hash of name prefix up to and including this component
} NameCompRecord;

/** \brief TLV Name
 *
 *  This struct stores indices of first \p NAME_MAX_INDEXED_COMPS name components, but permits
 *  arbitrary number of name components.
 */
typedef struct Name1
{
  uint16_t nOctets;   ///< TLV-LENGTH of Name element
  uint16_t nComps;    ///< number of components
  bool hasDigestComp; ///< ends with digest component?

  bool hasPrefixHashes; ///< (private) are comps[i].hash computed?
  NameCompRecord comps[NAME_MAX_INDEXED_COMPS]; ///< (private) components
} Name1;
static_assert(sizeof(Name1) <= 6 * RTE_CACHE_LINE_SIZE, "");

/** \brief Decode a name.
 *  \param[out] n the name.
 *  \retval NdnError_NameHasComponentAfterDigest unexpected non-digest component after digest.
 */
NdnError DecodeName1(TlvDecodePos* d, Name1* n);

void __Name1_GetComp_PastIndexed(const Name1* n, uint16_t i, TlvElement* ele);

/** \brief Get position of i-th name component.
 *  \param i name component index; <tt>0 <= i < nComps</tt>.
 *  \param[out] pos start position of i-th name component.
 *  \note pos->rem ends at the end of Name TLV.
 */
static void
Name1_GetCompPos(const Name1* n, uint16_t i, MbufLoc* pos)
{
  assert(i < n->nComps);

  if (likely(i < NAME_MAX_INDEXED_COMPS)) {
    return MbufLoc_Copy(pos, &n->comps[i].pos);
  }

  TlvElement ele;
  __Name1_GetComp_PastIndexed(n, i, &ele);
  MbufLoc_Copy(pos, &ele.first);
}

/** \brief Parse i-th name component.
 *  \param i name component index, must be less than n->nComps.
 *  \param[out] ele the element.
 */
static void
Name1_GetComp(const Name1* n, uint16_t i, TlvElement* ele)
{
  assert(i < n->nComps);

  if (unlikely(i >= NAME_MAX_INDEXED_COMPS)) {
    return __Name1_GetComp_PastIndexed(n, i, ele);
  }

  TlvDecodePos d;
  MbufLoc_Copy(&d, &n->comps[i].pos);
  NdnError e = DecodeTlvElement(&d, ele);
  assert(e == NdnError_OK); // cannot error in valid name
}

/** \brief Get size (in octets) of prefix with i components.
 */
static uint16_t
Name1_GetPrefixSize(const Name1* n, uint16_t i)
{
  assert(i <= n->nComps);

  if (i == 0) {
    return 0;
  }
  if (i == n->nComps) {
    return n->nOctets;
  }

  MbufLoc pos;
  Name1_GetCompPos(n, i, &pos);
  return MbufLoc_FastDiff(&n->comps[0].pos, &pos);
}

void __Name1_ComputePrefixHashes(Name1* n);
uint64_t __Name1_ComputePrefixHash_PastIndexed(const Name1* n, uint16_t i);

/** \brief Compute hash for prefix with i components.
 */
static uint64_t
Name1_ComputePrefixHash(const Name1* n, uint16_t i)
{
  if (i == 0) {
    return NAMEHASH_EMPTYHASH;
  }

  assert(i <= n->nComps);

  if (unlikely(i > NAME_MAX_INDEXED_COMPS)) {
    return __Name1_ComputePrefixHash_PastIndexed(n, i);
  }

  if (!n->hasPrefixHashes) {
    __Name1_ComputePrefixHashes((Name1*)n);
  }
  return n->comps[i - 1].hash;
}

/** \brief Compute hash for whole name.
 */
static uint64_t
Name1_ComputeHash(const Name1* n)
{
  return Name1_ComputePrefixHash(n, n->nComps);
}

/** \brief Compare two names.
 */
NameCompareResult Name1_Compare(const Name1* lhs, const Name1* rhs);

/** \brief Place name components in a linear buffer.
 *  \param scratch buffer space for copying name components when necessary.
 */
LName Name1_Linearize(const Name1* n, uint8_t scratch[NAME_MAX_LENGTH]);

#endif // NDN_DPDK_NDN_NAME1_H
