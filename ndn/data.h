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
} PData;

/** \brief Parse a packet as Data.
 *  \param[out] data the parsed Data packet.
 *  \param pkt the packet.
 *  \param nameMp mempool for allocating Name linearize mbufs,
 *                requires at least \p NAME_MAX_LENGTH dataroom.
 *  \retval NdnError_BadType packet is not Data.
 *  \retval NdnError_AllocError unable to allocate mbuf.
 */
NdnError PData_FromPacket(PData* data, struct rte_mbuf* pkt,
                          struct rte_mempool* nameMp);

#endif // NDN_DPDK_NDN_DATA_H
