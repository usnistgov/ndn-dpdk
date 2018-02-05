#ifndef NDN_DPDK_CONTAINER_FIB_ENTRY_H
#define NDN_DPDK_CONTAINER_FIB_ENTRY_H

/// \file

#include "../../iface/face.h"

#define FIB_ENTRY_MAX_NAME_LEN 500

#define FIB_ENTRY_MAX_NEXTHOPS 8

typedef struct FibEntry
{
  uint16_t nameL;    ///< TLV-LENGTH of name
  uint8_t nComps;    ///< number of name components
  uint8_t nNexthops; ///< number of nexthops

  /** \brief maximum potential LPM match relative to this entry
   *
   *  This field is known as '(MD - M)' in 2-stage LPM paper.
   *  This number must be no less than the depth of all FIB entries whose name starts
   *  with the name of this FIB entry, minus the depth of this entry.
   *  'depth' means number of name components.
   */
  uint8_t maxDepth;

  FaceId nexthops[FIB_ENTRY_MAX_NEXTHOPS];
  uint8_t nameV[FIB_ENTRY_MAX_NAME_LEN];
} FibEntry;

// FibEntry.nComps must be able to represent maximum number of name components that
// can fit in FIB_ENTRY_MAX_NAME_LEN octets.
static_assert(UINT8_MAX >= FIB_ENTRY_MAX_NAME_LEN / 2, "");

#endif // NDN_DPDK_CONTAINER_FIB_ENTRY_H
