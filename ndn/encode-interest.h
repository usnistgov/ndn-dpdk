#ifndef NDN_DPDK_NDN_ENCODE_INTEREST_H
#define NDN_DPDK_NDN_ENCODE_INTEREST_H

/// \file

#include "../core/pcg_basic.h"
#include "interest.h"

typedef struct NonceGen
{
  pcg32_random_t rng;
} NonceGen;

void
NonceGen_Init(NonceGen* g);

static uint32_t
NonceGen_Next(NonceGen* g)
{
  return pcg32_random_r(&g->rng);
}

/** \brief Template for Interest encoding.
 */
typedef struct InterestTemplate
{
  LName namePrefix;  ///< first part of the name
  uint32_t lifetime; ///< InterestLifetime in millis
  uint8_t hopLimit;  ///< HopLimit value
  bool canBePrefix;
  bool mustBeFresh;
  const uint8_t* fhV; ///< ForwardingHint TLV-VALUE
  uint16_t fhL;       ///< ForwardingHint TLV-LENGTH

  uint16_t bufferSize; ///< (pvt) used buffer size after \c bufferOff
  uint16_t bufferOff;  ///< (pvt) start offset within buffer
  uint16_t nonceOff;   ///< (pvt) nonce offset within buffer
} InterestTemplate;

uint16_t
__InterestTemplate_Prepare(InterestTemplate* tpl,
                           uint8_t* buffer,
                           uint16_t bufferSize,
                           const uint8_t* fhV);

/** \brief Prepare a buffer of "middle" fields.
 *  \param[out] buffer buffer space for CanBePrefix, MustBeFresh, ForwardingHint,
 *                     Nonce, InterestLifetime, HopLimit fields.
 *  \return 0 if success, otherwise a positive number indicates the required buffer size
 */
static uint16_t
InterestTemplate_Prepare(InterestTemplate* tpl,
                         uint8_t* buffer,
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
__EncodeInterest_GetTailroom(uint16_t bufferSize,
                             uint16_t nameL,
                             uint16_t paramL)
{
  return 1 + 3 + nameL +              // Name
         bufferSize + 1 + 5 + paramL; // Parameters
}

static uint16_t
EncodeInterest_GetTailroom(const InterestTemplate* tpl,
                           uint16_t nameSuffixL,
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

void
__EncodeInterest(struct rte_mbuf* m,
                 const InterestTemplate* tpl,
                 uint8_t* preparedBuffer,
                 uint16_t nameSuffixL,
                 const uint8_t* nameSuffixV,
                 uint32_t nonce,
                 uint16_t paramL,
                 const uint8_t* paramV,
                 const uint8_t* namePrefixV);

/** \brief Encode an Interest.
 *  \param m output mbuf, must be empty and is the only segment, must have
 *           \c EncodeInterest_GetHeadroom() in headroom and
 *           <tt>EncodeInterest_GetTailroom(tpl, nameSuffix.length, paramL)</tt> in tailroom;
 *           headroom for Ethernet and NDNLP headers shall be included if needed.
 *  \param preparedBuffer a buffer prepared with \c InterestTemplate_Prepare;
 *                        concurrent calls to this function must be distinct preparedBuffer;
 *                        the buffer may be duplicated after preparing if needed.
 *  \param nameSuffix second part of the name, set length to zero if not needed
 *  \param paramL Parameters TLV-LENGTH
 *  \param paramV Parameters TLV-VALUE
 */
static void
EncodeInterest(struct rte_mbuf* m,
               const InterestTemplate* tpl,
               uint8_t* preparedBuffer,
               LName nameSuffix,
               uint32_t nonce,
               uint16_t paramL,
               const uint8_t* paramV)
{
  __EncodeInterest(m,
                   tpl,
                   preparedBuffer,
                   nameSuffix.length,
                   nameSuffix.value,
                   nonce,
                   paramL,
                   paramV,
                   tpl->namePrefix.value);
}

#endif // NDN_DPDK_NDN_ENCODE_INTEREST_H
