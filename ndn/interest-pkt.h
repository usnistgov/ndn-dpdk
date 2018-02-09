#ifndef NDN_DPDK_NDN_INTEREST_PKT_H
#define NDN_DPDK_NDN_INTEREST_PKT_H

/// \file

#include "name.h"

/** \brief maximum number of forwarding hints
 */
#define INTEREST_MAX_FORWARDING_HINTS 1

#define DEFAULT_INTEREST_LIFETIME 4000

/** \brief TLV Interest
 */
typedef struct InterestPkt
{
  Name name;
  Name fwHints[INTEREST_MAX_FORWARDING_HINTS];
  MbufLoc nonce;     ///< start position and boundary of Nonce TLV-VALUE
  uint32_t lifetime; ///< InterestLifetime in millis
  uint8_t nFwHints;  ///< number of forwarding hints decoded in .fwHints
  bool mustBeFresh;  ///< has MustBeFresh?
} InterestPkt;

/** \brief Decode an Interest.
 *  \param[out] interest the Interest.
 *  \note Selectors other than MustBeFresh are silently ignored.
 *  \note Forwarding hints in excess of INTEREST_MAX_FORWARDING_HINTS are silently ignored.
 */
NdnError DecodeInterest(TlvDecoder* d, InterestPkt* interest);

/** \brief Get the Nonce, interpreted as little endian.
 */
static uint32_t
InterestPkt_GetNonce(const InterestPkt* interest)
{
  MbufLoc ml;
  MbufLoc_Copy(&ml, &interest->nonce);

  rte_le32_t nonce;
  bool ok = MbufLoc_ReadU32(&ml, &nonce);
  assert(ok);
  return rte_le_to_cpu_32(nonce);
}

void InterestPkt_SetNonce(InterestPkt* interest, uint32_t nonce);

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

#endif // NDN_DPDK_NDN_INTEREST_PKT_H