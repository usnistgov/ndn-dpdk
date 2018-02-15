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
 *  \param[out] interest the parsed Interest packet.
 *  \param pkt the packet.
 *  \param mpName mempool for allocating Name linearize mbufs,
 *                requires at least \p NAME_MAX_LENGTH dataroom.
 *  \retval NdnError_BadType packet is not Interest.
 *  \retval NdnError_AllocError unable to allocate mbuf.
 */
NdnError PInterest_FromPacket(PInterest* interest, struct rte_mbuf* pkt,
                              struct rte_mempool* mpName);

/** \brief Template for Interest encoding.
 */
typedef struct InterestTemplate
{
  LName namePrefix;  ///< first part of the name
  uint32_t lifetime; ///< InterestLifetime in millis
  HopLimit hopLimit; ///< HopLimit value, or \p HOP_LIMIT_OMITTED
  bool canBePrefix;
  bool mustBeFresh;
  const uint8_t* fhV; ///< ForwardingHint TLV-VALUE
  uint16_t fhL;       ///< ForwardingHint TLV-LENGTH

  uint16_t bufferSize; ///< (pvt) used buffer size after \p bufferOff
  uint16_t bufferOff;  ///< (pvt) start offset within buffer
  uint16_t nonceOff;   ///< (pvt) nonce offset within buffer
} InterestTemplate;

uint16_t __InterestTemplate_Prepare(InterestTemplate* tpl, uint8_t* buffer,
                                    uint16_t bufferSize, const uint8_t* fhV);

/** \brief Prepare a buffer of "middle" fields.
 *  \param[out] buffer buffer space for CanBePrefix, MustBeFresh, ForwardingHint,
 *                     Nonce, InterestLifetime, HopLimit fields.
 *  \return 0 if success, otherwise a positive number indicates the required buffer size
 */
static uint16_t
InterestTemplate_Prepare(InterestTemplate* tpl, uint8_t* buffer,
                         uint16_t bufferSize)
{
  return __InterestTemplate_Prepare(tpl, buffer, bufferSize, tpl->fhV);
}

static uint16_t
EncodeInterest_GetHeadroom()
{
  return 1 + 5; // Interest TL
}

static uint16_t
__EncodeInterest_GetTailroom(uint16_t bufferSize, uint16_t nameL,
                             uint16_t paramL)
{
  return 1 + 3 + nameL +              // Name
         bufferSize + 1 + 5 + paramL; // Parameters
}

static uint16_t
EncodeInterest_GetTailroom(const InterestTemplate* tpl, uint16_t nameSuffixL,
                           uint16_t paramL)
{
  return __EncodeInterest_GetTailroom(
    tpl->bufferSize, tpl->namePrefix.length + nameSuffixL, paramL);
}

/** \brief Get required tailroom for EncodeInterest output mbuf, assuming
 *         max name length, one delegation in forwarding hint, no parameters.
 */
static uint16_t
EncodeInterest_GetTailroomMax()
{
  const uint16_t maxBufferSize = 1 + 1 +                   // CanBePrefix
                                 1 + 1 +                   // MustBeFresh
                                 1 + 3 +                   // ForwardingHint
                                 1 + 3 +                   // FH.Delegation TL
                                 1 + 1 + 4 +               // FH.D.Preference
                                 1 + 3 + NAME_MAX_LENGTH + // FH.D.Name
                                 1 + 1 + 4 +               // Nonce
                                 1 + 1 + 4 +               // InterestLifetime
                                 1 + 1 + 1;                // HopLimit
  return __EncodeInterest_GetTailroom(maxBufferSize, NAME_MAX_LENGTH, 0);
}

void __EncodeInterest(struct rte_mbuf* m, const InterestTemplate* tpl,
                      uint8_t* preparedBuffer, uint16_t nameSuffixL,
                      const uint8_t* nameSuffixV, uint16_t paramL,
                      const uint8_t* paramV, const uint8_t* namePrefixV);

/** \brief Encode an Interest.
 *  \param m output mbuf, must be empty and is the only segment, must have
 *           \p EncodeInterest_GetHeadroom() in headroom and
 *           \p EncodeInterest_GetTailroom(tpl, nameSuffix.length, paramL) in tailroom;
 *           headroom for Ethernet and NDNLP headers shall be included if needed.
 *  \param preparedBuffer a buffer prepared with \p InterestTemplate_Prepare;
 *                        concurrent calls to this function must be distinct preparedBuffer;
 *                        the buffer may be duplicated after preparing if needed.
 *  \param nameSuffix second part of the name, set length to zero if not needed
 *  \param paramL Parameters TLV-LENGTH
 *  \param paramV Parameters TLV-VALUE
 */
static void
EncodeInterest(struct rte_mbuf* m, const InterestTemplate* tpl,
               uint8_t* preparedBuffer, LName nameSuffix, uint16_t paramL,
               const uint8_t* paramV)
{
  __EncodeInterest(m, tpl, preparedBuffer, nameSuffix.length, nameSuffix.value,
                   paramL, paramV, tpl->namePrefix.value);
}

#endif // NDN_DPDK_NDN_INTEREST_H
