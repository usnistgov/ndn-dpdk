#ifndef NDN_DPDK_NDN_DATA_H
#define NDN_DPDK_NDN_DATA_H

/// \file

#include "name.h"

/** \brief Parsed Data packet.
 */
typedef struct PData
{
  Name name;
  uint32_t freshnessPeriod; ///< FreshnessPeriod in millis
  uint32_t size;            ///< size of Data TLV

  bool hasDigest;
  uint8_t digest[32];
} PData;

/** \brief Parse a packet as Data.
 *  \param[out] data the parsed Data packet.
 *  \param pkt the packet.
 *  \param nameMp mempool for allocating Name linearize mbufs,
 *                requires at least \c NameMaxLength dataroom.
 *  \retval NdnErrBadType packet is not Data.
 *  \retval NdnErrAllocError unable to allocate mbuf.
 */
NdnError
PData_FromPacket(PData* data, struct rte_mbuf* pkt, struct rte_mempool* nameMp);

/** \brief Determine whether Data can satisfy Interest.
 */
DataSatisfyResult
PData_CanSatisfy(PData* data, PInterest* interest);

/** \brief Prepare a crypto_op for Data digest computation.
 *  \param npkt Data packet.
 *  \param[out] op an allocated crypto_op; will be populated but not enqueued.
 */
void
DataDigest_Prepare(Packet* npkt, struct rte_crypto_op* op);

/** \brief Finish Data digest computation.
 *  \param op a dequeued crypto_op; will be freed.
 *  \return the Data packet, or NULL if crypto_op was unsuccessful.
 */
Packet*
DataDigest_Finish(struct rte_crypto_op* op);

/** \brief Data encoder optimized for traffic generator.
 */
typedef struct DataGen
{
} DataGen;

/** \brief Prepare DataGen template.
 *  \param m template mbuf, must be empty and is the only segment, must have
 *           \c DataEstimatedTailroom in tailroom. DataGen takes ownership of this mbuf.
 */
DataGen*
DataGen_New(struct rte_mbuf* m,
            uint16_t nameSuffixL,
            const uint8_t* nameSuffixV,
            uint32_t freshnessPeriod,
            uint16_t contentL,
            const uint8_t* contentV);

void
DataGen_Close(DataGen* gen);

void
DataGen_Encode_(DataGen* gen,
                struct rte_mbuf* seg0,
                struct rte_mbuf* seg1,
                uint16_t namePrefixL,
                const uint8_t* namePrefixV);

/** \brief Encode Data with DataGen template.
 *  \param seg0 segment 0 mbuf, must be empty and is the only segment, must
 *              have \c DataEstimatedHeadroom in headroom \c DataEstimatedTailroom in tailroom.
 *              This becomes the encoded Data packet.
 *  \param seg1 segment 1 indirect mbuf. This is chained onto \p seg0 .
 */
static inline void
DataGen_Encode(DataGen* gen,
               struct rte_mbuf* seg0,
               struct rte_mbuf* seg1,
               LName namePrefix)
{
  DataGen_Encode_(gen, seg0, seg1, namePrefix.length, namePrefix.value);
}

#endif // NDN_DPDK_NDN_DATA_H
