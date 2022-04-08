#ifndef NDNDPDK_NDNI_LP_H
#define NDNDPDK_NDNI_LP_H

/** @file */

#include "common.h"

/** @brief NDNLPv2 layer 2 fields and reassembler state. */
typedef struct LpL2
{
  uint64_t seqNumBase; ///< seqNum-fragIndex
  uint8_t fragIndex;
  uint8_t fragCount;

  /**
   * @brief A bitmap of fragment arrival status.
   *
   * The bit (1 << i) indicates whether the i-th fragment has arrived.
   * 0 means it has arrived, 1 means it is still missing.
   * Bits of non-existent fragments are initialized to 0.
   * Thus, when this variable becomes zero, all the fragments have arrived.
   */
  uint32_t reassBitmap;
  struct cds_list_head reassNode;
  Packet* reassFrags[LpMaxFragments];
} LpL2;
static_assert(LpMaxFragments <= UINT8_MAX, "");

static __rte_always_inline uint64_t
LpL2_GetSeqNum(const LpL2* l2)
{
  return l2->seqNumBase + l2->fragIndex;
}

typedef struct LpPitToken
{
  uint8_t length;
  uint8_t value[32];
} __rte_packed LpPitToken;

__attribute__((nonnull)) static __rte_always_inline bool
LpPitToken_Equal(const LpPitToken* a, const LpPitToken* b)
{
  return memcmp(a, b, sizeof(*a)) == 0;
}

__attribute__((nonnull)) static __rte_always_inline void
LpPitToken_Set(LpPitToken* token, uint8_t length, const uint8_t* value)
{
  token->length = length;
  rte_memcpy(token->value, value, length);
  memset(RTE_PTR_ADD(token->value, length), 0, sizeof(token->value) - length);
}

/** @brief printf format string for LpPitToken. */
#define PRI_LpPitToken                                                                             \
  "%02" PRIx8 "%02" PRIx8 "%02" PRIx8 "%02" PRIx8 "%02" PRIx8 "%02" PRIx8 "%02" PRIx8 "%02" PRIx8  \
  "%02" PRIx8 "%02" PRIx8 "%02" PRIx8 "%02" PRIx8 "%02" PRIx8 "%02" PRIx8 "%02" PRIx8 "%02" PRIx8  \
  "%02" PRIx8 "%02" PRIx8 "%02" PRIx8 "%02" PRIx8 "%02" PRIx8 "%02" PRIx8 "%02" PRIx8 "%02" PRIx8  \
  "%02" PRIx8 "%02" PRIx8 "%02" PRIx8 "%02" PRIx8 "%02" PRIx8 "%02" PRIx8 "%02" PRIx8 "%02" PRIx8  \
  "(%" PRIu8 ")"

/** @brief printf variables for LpPitToken. */
#define LpPitToken_Fmt(token)                                                                      \
  (token)->value[0], (token)->value[1], (token)->value[2], (token)->value[3], (token)->value[4],   \
    (token)->value[5], (token)->value[6], (token)->value[7], (token)->value[8], (token)->value[9], \
    (token)->value[10], (token)->value[11], (token)->value[12], (token)->value[13],                \
    (token)->value[14], (token)->value[15], (token)->value[16], (token)->value[17],                \
    (token)->value[18], (token)->value[19], (token)->value[20], (token)->value[21],                \
    (token)->value[22], (token)->value[23], (token)->value[24], (token)->value[25],                \
    (token)->value[26], (token)->value[27], (token)->value[28], (token)->value[29],                \
    (token)->value[30], (token)->value[31], (token)->length

/** @brief NDNLPv2 layer 3 fields. */
typedef struct LpL3
{
  uint8_t nackReason;
  uint8_t congMark;
  LpPitToken pitToken;
} LpL3;

/** @brief Parsed NDNLPv2 header. */
typedef struct LpHeader
{
  LpL3 l3;
  LpL2 l2;
} LpHeader;

/**
 * @brief Parse NDNLPv2 header and strip from mbuf.
 * @param pkt a uniquely owned, unsegmented, direct mbuf.
 * @return whether success.
 * @post @p pkt contains only (fragment of) network layer packet.
 *
 * This function recognizes these NDNLPv2 features:
 * @li indexed fragmentation-reassembly
 * @li PIT token
 * @li network nack
 * @li congestion mark
 *
 * This function does not check whether header fields are applicable to network layer packet type,
 * because network layer type is unknown before reassembly. For example, it would accept Nack
 * header on Data packet.
 */
__attribute__((nonnull)) bool
LpHeader_Parse(LpHeader* lph, struct rte_mbuf* pkt);

/**
 * @brief Prepend NDNLPv2 header to mbuf.
 * @param pkt target mbuf, must have enough headroom.
 * @pre @p pkt contains (fragment of) network layer packet.
 * @post @p pkt contains LpPacket.
 */
__attribute__((nonnull)) void
LpHeader_Prepend(struct rte_mbuf* pkt, const LpL3* l3, const LpL2* l2);

#endif // NDNDPDK_NDN_LP_H
