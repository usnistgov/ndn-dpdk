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
 *                requires at least \c NAME_MAX_LENGTH dataroom.
 *  \retval NdnError_BadType packet is not Data.
 *  \retval NdnError_AllocError unable to allocate mbuf.
 */
NdnError PData_FromPacket(PData* data, struct rte_mbuf* pkt,
                          struct rte_mempool* nameMp);

typedef struct Packet Packet;

/** \brief Prepare a crypto_op for Data digest computation.
 *  \param npkt Data packet.
 *  \param[out] op an allocated crypto_op; will be populated but not enqueued.
 */
void DataDigest_Prepare(Packet* npkt, struct rte_crypto_op* op);

/** \brief Finish Data digest computation.
 *  \param op a dequeued crypto_op; will be freed.
 *  \return the Data packet, or NULL if crypto_op was unsuccessful.
 */
Packet* DataDigest_Finish(struct rte_crypto_op* op);

#endif // NDN_DPDK_NDN_DATA_H
