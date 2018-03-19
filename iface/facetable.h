#ifndef NDN_DPDK_IFACE_FACETABLE_H
#define NDN_DPDK_IFACE_FACETABLE_H

/// \file

#include "face.h"

/** \brief Table of faces indexed by FaceId.
 *
 *  This data structure is thread-safe.
 */
typedef struct FaceTable
{
  atomic_int count;
  Face* _Atomic table[FACEID_MAX];
} FaceTable;

/** \brief Count faces in the FaceTable.
 */
int FaceTable_Count(FaceTable* ft);

/** \brief Get face with specified FaceId.
 */
Face* FaceTable_GetFace(FaceTable* ft, FaceId id);

/** \brief Add face with pre-assigned FaceId.
 *  \pre face->id != FACEID_INVALID
 *  \pre FaceTable_GetFace(ft, face->id) == NULL
 */
void FaceTable_AddFace(FaceTable* ft, Face* face);

/** \brief Remove face with specified FaceId.
 *  \post If face exists, \p Face_Close is invoked.
 */
void FaceTable_RemoveFace(FaceTable* ft, FaceId id);

#endif // NDN_DPDK_IFACE_FACETABLE_H
