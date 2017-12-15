#ifndef NDN_TRAFFIC_DPDK_NDN_INTEREST_H
#define NDN_TRAFFIC_DPDK_NDN_INTEREST_H

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
  MbufLoc nonce;     ///< start position of Nonce TLV-VALUE
  uint32_t lifetime; ///< InterestLifetime in mills
  uint8_t nFwHints;  ///< number of forwarding hints decoded in .fwHints
  bool mustBeFresh;  ///< has MustBeFresh?
} InterestPkt;
static_assert(sizeof(Name) <= 4 * RTE_CACHE_LINE_SIZE, "");

/** \brief Decode an Interest.
 *  \param[out] interest the Interest.
 *  \note Selectors other than MustBeFresh are silently ignored.
 *  \note Forwarding hints in excess of INTEREST_MAX_FORWARDING_HINTS are silently ignored.
 */
NdnError DecodeInterest(TlvDecoder* d, InterestPkt* interest, size_t* len);

/** \brief Get the Nonce in network byte order.
 */
static inline uint32_t
InterestPkt_GetNonce(const InterestPkt* interest)
{
  MbufLoc ml;
  MbufLoc_Copy(&ml, &interest->nonce);

  uint32_t nonce;
  bool ok = MbufLoc_ReadU32(&ml, &nonce);
  assert(ok);
  return nonce;
}

void InterestPkt_SetNonce(InterestPkt* interest, uint32_t nonce);

#endif // NDN_TRAFFIC_DPDK_NDN_INTEREST_H