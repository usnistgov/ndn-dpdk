#ifndef NDN_DPDK_NDN_NAME_H
#define NDN_DPDK_NDN_NAME_H

/// \file

#include "namehash.h"
#include "tlv-element.h"

/** \brief maximum supported name length (TLV-LENGTH of Name element)
 */
#define NAME_MAX_LENGTH 2048

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
typedef struct Name
{
  uint16_t nOctets;   ///< TLV-LENGTH of Name element
  uint16_t nComps;    ///< number of components
  bool hasDigestComp; ///< ends with digest component?

  bool hasPrefixHashes; ///< (private) are comps[i].hash computed?
  NameCompRecord comps[NAME_MAX_INDEXED_COMPS]; ///< (private) components
} Name;
static_assert(sizeof(Name) <= 6 * RTE_CACHE_LINE_SIZE, "");

/** \brief Decode a name.
 *  \param[out] n the name.
 *  \retval NdnError_NameHasComponentAfterDigest unexpected non-digest component after digest.
 */
NdnError DecodeName(TlvDecoder* d, Name* n);

void __Name_GetComp_PastIndexed(const Name* n, uint16_t i, TlvElement* ele);

/** \brief Get position of i-th name component.
 *  \param i name component index; <tt>0 <= i < nComps</tt>.
 *  \param[out] pos start position of i-th name component.
 *  \note pos->rem ends at the end of Name TLV.
 */
static void
Name_GetCompPos(const Name* n, uint16_t i, MbufLoc* pos)
{
  assert(i < n->nComps);

  if (likely(i < NAME_MAX_INDEXED_COMPS)) {
    return MbufLoc_Copy(pos, &n->comps[i].pos);
  }

  TlvElement ele;
  __Name_GetComp_PastIndexed(n, i, &ele);
  MbufLoc_Copy(pos, &ele.first);
}

/** \brief Parse i-th name component.
 *  \param i name component index, must be less than n->nComps.
 *  \param[out] ele the element.
 */
static void
Name_GetComp(const Name* n, uint16_t i, TlvElement* ele)
{
  assert(i < n->nComps);

  if (unlikely(i >= NAME_MAX_INDEXED_COMPS)) {
    return __Name_GetComp_PastIndexed(n, i, ele);
  }

  TlvDecoder d;
  MbufLoc_Copy(&d, &n->comps[i].pos);
  NdnError e = DecodeTlvElement(&d, ele);
  assert(e == NdnError_OK); // cannot error in valid name
}

/** \brief Get size (in octets) of prefix with i components.
 */
static uint16_t
Name_GetPrefixSize(const Name* n, uint16_t i)
{
  assert(i <= n->nComps);

  if (i == 0) {
    return 0;
  }
  if (i == n->nComps) {
    return n->nOctets;
  }

  MbufLoc pos;
  Name_GetCompPos(n, i, &pos);
  return MbufLoc_FastDiff(&n->comps[0].pos, &pos);
}

void __Name_ComputePrefixHashes(Name* n);
uint64_t __Name_ComputePrefixHash_PastIndexed(const Name* n, uint16_t i);

/** \brief Compute hash for prefix with i components.
 */
static uint64_t
Name_ComputePrefixHash(const Name* n, uint16_t i)
{
  if (i == 0) {
    return NAMEHASH_EMPTYHASH;
  }

  assert(i <= n->nComps);

  if (unlikely(i > NAME_MAX_INDEXED_COMPS)) {
    return __Name_ComputePrefixHash_PastIndexed(n, i);
  }

  if (!n->hasPrefixHashes) {
    __Name_ComputePrefixHashes((Name*)n);
  }
  return n->comps[i - 1].hash;
}

/** \brief Compute hash for whole name.
 */
static uint64_t
Name_ComputeHash(const Name* n)
{
  return Name_ComputePrefixHash(n, n->nComps);
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

/** \brief Name in linear buffer.
 */
typedef struct LName
{
  const uint8_t* value;
  uint16_t length;
} LName;

/** \brief Place name components in a linear buffer.
 *  \param scratch buffer space for copying name components when necessary.
 */
LName Name_Linearize(const Name* n, uint8_t scratch[NAME_MAX_LENGTH]);

uint64_t LName_ComputeHash(LName n);

/** \brief Compare two names in linear buffers.
 */
NameCompareResult LName_Compare(LName lhs, LName rhs);

#endif // NDN_DPDK_NDN_NAME_H