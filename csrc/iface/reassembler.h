#ifndef NDNDPDK_IFACE_REASSEMBLER_H
#define NDNDPDK_IFACE_REASSEMBLER_H

/** @file */

#include "common.h"

/** @brief NDNLPv2 reassembler. */
typedef struct Reassembler
{
  uint64_t nDeliverPackets;   ///< delivered packets
  uint64_t nDeliverFragments; ///< delivered fragments
  uint64_t nDropFragments;    ///< dropped fragments

  struct rte_hash* table;
  TAILQ_HEAD(LpL2Queue, LpL2) list;
  uint32_t count;
  uint32_t capacity;
} Reassembler;

/**
 * @brief Create a reassembler.
 * @param reass zero Reassembler struct, usually embedded in a larger struct.
 * @param id memzone identifier, must be unique.
 * @param capacity maximum number of partial messages.
 *                 Oldest partial message is discarded when this limit is reached.
 * @param numaSocket where to allocate memory.
 *
 * Caller must invoke @p Pit_Init and @p Cs_Init to initialize each table.
 */
__attribute__((nonnull)) bool
Reassembler_New(Reassembler* reass, const char* id, uint32_t capacity, unsigned numaSocket);

/** @brief Release all memory except @p reass struct. */
__attribute__((nonnull)) void
Reassembler_Close(Reassembler* reass);

/**
 * @brief Accept an incoming fragment.
 * @param fragment an NDNLPv2 fragment. It must have type PktFragment and FragCount greater than 1.
 * @return a reassembled network layer packet, unparsed.
 * @retval NULL no network layer packet is ready.
 */
__attribute__((nonnull)) Packet*
Reassembler_Accept(Reassembler* reass, Packet* fragment);

#endif // NDNDPDK_IFACE_REASSEMBLER_H
