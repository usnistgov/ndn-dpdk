#ifndef NDNDPDK_NDNI_INTEREST_H
#define NDNDPDK_NDNI_INTEREST_H

/** @file */

#include "../vendor/pcg_basic.h"
#include "name.h"

/** @brief Random nonce generator. */
typedef struct NonceGen
{
  pcg32_random_t rng;
} NonceGen;

void
NonceGen_Init(NonceGen* g);

static __rte_always_inline uint32_t
NonceGen_Next(NonceGen* g)
{
  return pcg32_random_r(&g->rng);
}

/** @brief Parsed Interest packet. */
typedef struct PInterest
{
  uint32_t nonce;    ///< Nonce
  uint32_t lifetime; ///< InterestLifetime in millis

  uint32_t nonceOffset; ///< offset of Nonce within Interest TLV-VALUE
  uint8_t guiderSize;   ///< size of Nonce+InterestLifetime+HopLimit

  uint8_t hopLimit; ///< HopLimit value, "omitted" is same as 0xFF
#ifdef GODEF
  uint8_t placeholder0_;
#else
  struct
  {
    bool canBePrefix : 1;
    bool mustBeFresh : 1;
    uint8_t nFwHints : 3;    ///< number of forwarding hints, up to PInterestMaxFwHints
    int8_t activeFwHint : 3; ///< index of active forwarding hint
  } __rte_packed;
#endif

  PName name;
  const uint8_t* fwHintV[PInterestMaxFwHints]; ///< TLV-VALUE of forwarding hints
  uint16_t fwHintL[PInterestMaxFwHints];       ///< TLV-LENGTH of forwarding hints
  PName fwHint;                                ///< parsed forwarding hint at activeFwHint

  uint64_t diskSlot; ///< DiskStore slot number
  Packet* diskData;  ///< DiskStore loaded Data
} PInterest;

/**
 * @brief Parse Interest.
 * @param pkt a uniquely owned, possibly segmented, direct mbuf that contains Interest TLV.
 * @return whether success.
 */
__attribute__((nonnull)) bool
PInterest_Parse(PInterest* interest, struct rte_mbuf* pkt);

/**
 * @brief Retrieve i-th forwarding hint name.
 * @return whether success.
 * @pre i >= 0 && i < interest->nFwHints
 * @post interest->activeFwHint == i
 * @post interest->fwHint reflects i-th forwarding hint.
 */
__attribute__((nonnull)) bool
PInterest_SelectFwHint(PInterest* interest, int i);

/** @brief Interest guider fields. */
typedef struct InterestGuiders
{
  uint32_t nonce;
  uint32_t lifetime;
  uint8_t hopLimit;
} InterestGuiders;

/**
 * @brief printf format string for InterestGuiders.
 * @code
 * printf("mbuf=%p " PRI_InterestGuiders " suffix", mbuf, InterestGuiders_Fmt(guiders))
 * @endcode
 */
#define PRI_InterestGuiders "nonce=%08" PRIx32 " lifetime=%" PRIu32 " hopLimit=%" PRIu8

/**
 * @brief printf arguments for InterestGuiders.
 * @param g InterestGuiders instance.
 */
#define InterestGuiders_Fmt(g) (g).nonce, (g).lifetime, (g).hopLimit

/**
 * @brief Modify Interest guiders.
 * @param[in] npkt original Interest packet.
 * @return cloned and modified Interest packet.
 * @retval NULL allocation failure.
 */
__attribute__((nonnull)) Packet*
Interest_ModifyGuiders(Packet* npkt, InterestGuiders guiders, PacketMempools* mp,
                       PacketTxAlign align);

/** @brief Template for Interest encoding. */
typedef struct InterestTemplate
{
  uint16_t prefixL;                       ///< Name prefix length
  uint16_t midLen;                        ///< midBuf length
  uint16_t nonceVOffset;                  ///< Nonce TLV-VALUE offset within midBuf
  uint8_t prefixV[NameMaxLength];         ///< Name prefix
  uint8_t midBuf[InterestTemplateBufLen]; ///< fields after Name
} InterestTemplate;

/**
 * @brief Encode Interest with InterestTemplate.
 * @param m a uniquely owned, unsegmented, direct, empty mbuf.
 *          It must have @c InterestTemplateDataroom buffer size.
 * @return encoded packet, converted from @c m .
 */
__attribute__((nonnull, returns_nonnull)) Packet*
InterestTemplate_Encode(const InterestTemplate* tpl, struct rte_mbuf* m, LName suffix,
                        uint32_t nonce);

#endif // NDNDPDK_NDNI_INTEREST_H
