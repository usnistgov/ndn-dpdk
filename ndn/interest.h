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
  HOP_LIMIT_OMITTED = 0x0101, ///< HopLimit is omitted
  HOP_LIMIT_ZERO = 0x0100,    ///< HopLimit was zero before decrementing
};

/** \brief Parsed Interest packet.
 */
typedef struct PInterest
{
  uint32_t guiderOff;  ///< size of Name through ForwardingHint
  uint32_t guiderSize; ///< size of Nonce+InterestLifetime

  Name name;
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
 *  \param[out] interest the parsed Interest packet.
 *  \param pkt the packet.
 *  \param nameMp mempool for allocating Name linearize mbufs,
 *                requires at least \p NAME_MAX_LENGTH dataroom.
 *  \retval NdnError_BadType packet is not Interest.
 *  \retval NdnError_AllocError unable to allocate mbuf.
 */
NdnError PInterest_FromPacket(PInterest* interest, struct rte_mbuf* pkt,
                              struct rte_mempool* nameMp);

/** \brief Parse a forwarding hint.
 *  \param index forwarding hint index, must be less than \p interest->nFhs.
 *  \post interest->thisFhIndex == index
 *  \post interest->thisFh reflects the index-th forwarding hint.
 */
NdnError PInterest_ParseFh(PInterest* interest, uint8_t index);

static uint16_t
ModifyInterest_SizeofGuider()
{
  return 1 + 1 + 4 + // Nonce
         1 + 1 + 4;  // InterestLifetime
}

/** \brief Modify Interest nonce and lifetime.
 *  \param[in] npkt the original Interest packet;
 *             must have \p Packet_GetInterestHdr().
 *  \param headerMp mempool for storing Interest TL;
 *                  must have \p EncodeInterest_GetHeadroom() dataroom,
 *                  and must fulfill requirements of \p Packet_FromMbuf();
 *                  may have additional headroom for lower layer headers.
 *  \param guiderMp mempool for storing Nonce and InterestLifetime;
 *                  must have \p ModifyInterest_SizeofGuider() dataroom.
 *  \param indirectMp mempool for allocating indirect mbufs.
 *  \return cloned and modified packet that has \p Packet_GetInterestHdr().
 *  \retval NULL allocation failure.
 */
Packet* ModifyInterest(Packet* npkt, uint32_t nonce, uint32_t lifetime,
                       struct rte_mempool* headerMp,
                       struct rte_mempool* guiderMp,
                       struct rte_mempool* indirectMp);

#endif // NDN_DPDK_NDN_INTEREST_H
