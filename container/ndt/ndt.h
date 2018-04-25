#ifndef NDN_DPDK_CONTAINER_NDT_NDT_H
#define NDN_DPDK_CONTAINER_NDT_NDT_H

/// \file

#include "../../ndn/name.h"

/** \brief Per-thread counters for NDT.
 */
typedef struct NdtThread
{
  uint64_t nLookups;
  uint16_t nHits[0];
} NdtThread;

/** \brief The Name Dispatch Table (NDT).
 */
typedef struct Ndt
{
  _Atomic uint8_t* table;
  NdtThread** threads;
  uint64_t indexMask;
  uint64_t sampleMask;
  uint16_t prefixLen;
  uint8_t nThreads;
} Ndt;

/** \brief Initialize NDT.
 *  \param prefixLen prefix length for computing hash.
 *  \param indexBits how many bits to truncate the hash into table entry index.
 *  \param sampleFreq sample once every 2^sampleFreq lookups.
 *  \param nThreads number of lookup threads
 *  \param numaSockets array of \p nThreads elements indicating NUMA socket of each
 *                     lookup thread; numaSockets[0] will be used for the table
 *  \return array of threads
 */
NdtThread** Ndt_Init(Ndt* ndt, uint16_t prefixLen, uint8_t indexBits,
                     uint8_t sampleFreq, uint8_t nThreads,
                     const unsigned* numaSockets);

/** \brief Release all memory associated with NDT, except \p ndt itself.
 */
void Ndt_Close(Ndt* ndt);

/** \brief Access NdtThread struct.
 */
static NdtThread*
Ndt_GetThread(const Ndt* ndt, uint8_t id)
{
  assert(id < ndt->nThreads);
  return ndt->threads[id];
}

/** \brief Update NDT.
 *  \param hash a prefix hash mapped into the table entry.
 *  \param value new PIT partition number in the table entry.
 *  \return table index.
 */
uint64_t Ndt_Update(Ndt* ndt, uint64_t hash, uint8_t value);

/** \brief Query NDT.
 */
uint8_t Ndt_Lookup(const Ndt* ndt, NdtThread* ndtt, const PName* name,
                   const uint8_t* nameV);

#endif // NDN_DPDK_CONTAINER_NDT_NDT_H
