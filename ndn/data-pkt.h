#ifndef NDN_DPDK_NDN_DATA_PKT_H
#define NDN_DPDK_NDN_DATA_PKT_H

/// \file

#include "name.h"

/** \brief TLV Data
 */
typedef struct DataPkt
{
  Name name;
  MbufLoc content; ///< start position and boundary of Content TLV-VALUE
  uint32_t freshnessPeriod; ///< FreshnessPeriod in millis
} DataPkt;

/** \brief Decode a Data.
 *  \param[out] data the Data.
 */
NdnError DecodeData(TlvDecoder* d, DataPkt* data);

#endif // NDN_DPDK_NDN_DATA_PKT_H