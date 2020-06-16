#ifndef NDN_DPDK_NDN_INTEREST_H
#define NDN_DPDK_NDN_INTEREST_H

/// \file

#include "../core/pcg_basic.h"
#include "name.h"

/** \brief maximum number of forwarding hints
 */
#define INTEREST_MAX_FHS 4

#define DEFAULT_INTEREST_LIFETIME 4000

/** \brief Parsed Interest packet.
 */
typedef struct PInterest
{
  uint32_t nonce;    ///< Nonce interpreted as little endian
  uint32_t lifetime; ///< InterestLifetime in millis

  uint32_t guiderOff;  ///< size of Name through ForwardingHint
  uint16_t guiderSize; ///< size of Nonce+InterestLifetime+HopLimit

  uint8_t hopLimit; ///< HopLimit value, "omitted" is same as 0xFF
  struct
  {
    bool canBePrefix : 1;
    bool mustBeFresh : 1;
    uint8_t nFhs : 3;    ///< number of fwhints, up to INTEREST_MAX_FHS
    int8_t activeFh : 3; ///< index of active fwhint, -1 for none
  } __rte_packed;

  Name name;

  const uint8_t* fhNameV[INTEREST_MAX_FHS];
  uint16_t fhNameL[INTEREST_MAX_FHS];
  Name activeFhName; ///< a parsed forwarding hint at index \c activeFh

  uint64_t diskSlotId; ///< DiskStore slot number
  Packet* diskData;    ///< DiskStore loaded Data
} PInterest;

/** \brief Parse a packet as Interest.
 *  \param[out] interest the parsed Interest packet.
 *  \param pkt the packet.
 *  \param nameMp mempool for allocating Name linearize mbufs,
 *                requires at least \c NAME_MAX_LENGTH dataroom.
 *  \retval NdnErrBadType packet is not Interest.
 *  \retval NdnErrAllocError unable to allocate mbuf.
 */
NdnError
PInterest_FromPacket(PInterest* interest,
                     struct rte_mbuf* pkt,
                     struct rte_mempool* nameMp);

/** \brief Set active forwarding hint.
 *  \param index fwhint index, must be less than \c interest->nFhs, or -1 for none.
 *  \post interest->activeFh == index
 *  \post interest->activeFhName reflects the index-th fwhint.
 */
NdnError
PInterest_SelectActiveFh(PInterest* interest, int8_t index);

/** \brief Random nonce generator.
 */
typedef struct NonceGen
{
  pcg32_random_t rng;
} NonceGen;

void
NonceGen_Init(NonceGen* g);

static inline uint32_t
NonceGen_Next(NonceGen* g)
{
  return pcg32_random_r(&g->rng);
}

/** \brief Modify Interest nonce and lifetime.
 *  \param[in] npkt the original Interest packet;
 *                  must have \c Packet_GetInterestHdr().
 *  \return cloned and modified packet that has \c Packet_GetInterestHdr().
 *  \retval NULL allocation failure.
 */
Packet*
ModifyInterest(Packet* npkt,
               uint32_t nonce,
               uint32_t lifetime,
               uint8_t hopLimit,
               struct rte_mempool* headerMp,
               struct rte_mempool* guiderMp,
               struct rte_mempool* indirectMp);

#define INTEREST_TEMPLATE_BUFLEN (2 * NAME_MAX_LENGTH + 256)

/** \brief Template for Interest encoding.
 */
typedef struct InterestTemplate
{
  uint16_t prefixL;                         ///< Name prefix length
  uint16_t midLen;                          ///< midBuffer length
  uint16_t nonceOff;                        ///< NonceV offset within midBuffer
  uint8_t prefixV[NAME_MAX_LENGTH];         ///< Name prefix
  uint8_t midBuf[INTEREST_TEMPLATE_BUFLEN]; ///< "middle" field
} InterestTemplate;

void
EncodeInterest_(struct rte_mbuf* m,
                const InterestTemplate* tpl,
                uint16_t suffixL,
                const uint8_t* suffixV,
                uint32_t nonce);

/** \brief Encode an Interest.
 */
static inline void
EncodeInterest(struct rte_mbuf* m,
               const InterestTemplate* tpl,
               LName suffix,
               uint32_t nonce)
{
  EncodeInterest_(m, tpl, suffix.length, suffix.value, nonce);
}

#endif // NDN_DPDK_NDN_INTEREST_H
