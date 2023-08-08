#ifndef NDNDPDK_IFACE_REASSEMBLER_H
#define NDNDPDK_IFACE_REASSEMBLER_H

/** @file */

#include "common.h"

/** @brief NDNLPv2 reassembler. */
typedef struct Reassembler {
  uint64_t nDeliverPackets;   ///< delivered packets
  uint64_t nDeliverFragments; ///< delivered fragments
  uint64_t nDropFragments;    ///< dropped fragments

  struct rte_hash* table;
  struct cds_list_head list;
  uint32_t count;
  uint32_t capacity;
} Reassembler;

/**
 * @brief Initialize a reassembler.
 * @param reass zero Reassembler struct, usually embedded in a larger struct.
 * @param id memzone identifier, must be unique.
 * @param capacity maximum number of partial messages.
 *                 Oldest partial message is discarded when this limit is reached.
 * @param numaSocket where to allocate memory.
 * @return whether success. Error code is in @c rte_errno .
 */
__attribute__((nonnull)) bool
Reassembler_Init(Reassembler* reass, const char* id, uint32_t capacity, int numaSocket);

/** @brief Release all memory except @p reass struct. */
__attribute__((nonnull)) void
Reassembler_Close(Reassembler* reass);

/**
 * @brief Accept an incoming fragment.
 * @param fragment an NDNLPv2 fragment. It must have type PktFragment and FragCount greater than 1.
 *                 Non-contiguous packet is unsupported and will be rejected.
 * @return a reassembled network layer packet, unparsed.
 * @retval NULL no network layer packet is ready.
 */
__attribute__((nonnull)) Packet*
Reassembler_Accept(Reassembler* reass, Packet* fragment);

#endif // NDNDPDK_IFACE_REASSEMBLER_H
