#ifndef NDN_DPDK_CONTAINER_PCCT_CS_STRUCT_H
#define NDN_DPDK_CONTAINER_PCCT_CS_STRUCT_H

/// \file

#include "common.h"

/** \brief The Content Store (CS).
 *
 *  Cs* is Pcct*.
 */
typedef struct Cs
{
} Cs;

/** \brief PCCT private data for CS.
 */
typedef struct CsPriv
{
  uint32_t capacity;
  uint32_t nEntries;
} CsPriv;

#endif // NDN_DPDK_CONTAINER_PCCT_CS_STRUCT_H
