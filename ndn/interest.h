#ifndef NDN_DPDK_NDN_INTEREST_H
#define NDN_DPDK_NDN_INTEREST_H

/// \file

#include "name.h"

/** \brief maximum number of forwarding hints
 */
#define INTEREST_MAX_FHS 4

#define DEFAULT_INTEREST_LIFETIME 4000

/** \brief Interest HopLimit field.
 */
typedef uint16_t HopLimit;
enum HopLimitSpecial
{
  HOP_LIMIT_OMITTED = 0x0100, ///< HopLimit is omitted
  HOP_LIMIT_ZERO = 0x0101,    ///< HopLimit was zero before decrementing
};

/** \brief Parsed Interest packet.
 */
typedef struct PInterest
{
  Name name;
  MbufLoc guiderLoc; ///< where are Nonce and InterestLifetime
  uint32_t nonce;    ///< Nonce interpreted as little endian
  uint32_t lifetime; ///< InterestLifetime in millis
  HopLimit hopLimit; ///< HopLimit value after decrementing, or HopLimitSpecial
  bool canBePrefix;
  bool mustBeFresh;
  uint8_t nFhs;       ///< number of forwarding hints in \p fh
  int8_t thisFhIndex; ///< index of current forwarding hint in \p thisFh, or -1
  LName fh[INTEREST_MAX_FHS];
  Name thisFh; ///< a parsed forwarding hint at index \p thisFhIndex
} PInterest;

/** \brief Parse a packet as Interest.
 *  \param[out] the parsed Interest packet.
 *  \param pkt the packet.
 *  \param mpName mempool for allocating Name linearize mbufs,
 *                requires at least \p NAME_MAX_LENGTH dataroom.
 *  \retval NdnError_BadType packet is not Interest.
 *  \retval NdnError_AllocError unable to allocate mbuf.
 */
NdnError PInterest_FromPacket(PInterest* interest, struct rte_mbuf* pkt,
                              struct rte_mempool* mpName);

typedef struct InterestTemplate
{
  const uint8_t* namePrefix;
  uint16_t namePrefixSize;
  const uint8_t* nameSuffix;
  uint16_t nameSuffixSize;
  bool mustBeFresh;
  uint32_t lifetime;
  const uint8_t* fwHints;
  uint16_t fwHintsSize;
} InterestTemplate;

static uint16_t
EncodeInterest_GetHeadroom()
{
  return 1 + 5; // Name TL
}

static uint16_t
EncodeInterest_GetTailroom(const InterestTemplate* tpl)
{
  return 1 + 5 + tpl->namePrefixSize + tpl->nameSuffixSize + // Name
         1 + 1 +                                             //Selectors
         1 + 1 +                                             // MustBeFresh
         1 + 1 + 4 +                                         // Nonce
         1 + 1 + 4 +                                         // InterestLifetime
         1 + 5 + tpl->fwHintsSize;                           // ForwardingHint
}

/** \brief Get required tailroom for EncodeInterest output mbuf,
 *         assuming max name length and one delegation in forwarding hint.
 */
static uint16_t
EncodeInterest_GetTailroomMax()
{
  return 1 + 5 + NAME_MAX_LENGTH + // Name
         1 + 1 +                   // Selectors TL
         1 + 1 +                   // S.MustBeFresh
         1 + 1 + 4 +               // Nonce
         1 + 1 + 4 +               // InterestLifetime
         1 + 5 +                   // ForwardingHint TL
         1 + 5 +                   // FH.Delegation TL
         1 + 1 + 4 +               // D.Preference
         1 + 5 + NAME_MAX_LENGTH;  // D.Name
}

// Golang cgocheck is unhappy if tpl->namePrefix etc points to Go memory.
void __EncodeInterest(struct rte_mbuf* m, const InterestTemplate* tpl,
                      const uint8_t* namePrefix, const uint8_t* nameSuffix,
                      const uint8_t* fwHints);

/** \brief Make an Interest.
 *  \param m output mbuf, must be empty and is the only segment, must have
 *           \p EncodeInterest_GetHeadroom() in headroom and
 *           \p EncodeInterest_GetTailroom(tpl) in tailroom;
 *           headroom for Ethernet and NDNLP headers shall be included if needed.
 *  \param tpl
 */
static void
EncodeInterest(struct rte_mbuf* m, const InterestTemplate* tpl)
{
  __EncodeInterest(m, tpl, tpl->namePrefix, tpl->nameSuffix, tpl->fwHints);
}

#endif // NDN_DPDK_NDN_INTEREST_H
