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

/** \brief Parse a Data from TlvElement.
 *  \param ele TLV Name element, TLV-TYPE must be TT_Data
 *  \retval NdnError_Fragmented Name TLV-VALUE is not in consecutive memory
 */
NdnError PData_FromElement(PData* data, const TlvElement* ele);

static uint16_t
EncodeData1_GetHeadroom()
{
  return 1 + 5 + // Data
         1 + 3;  // Name TL
}

static uint16_t
EncodeData1_GetTailroom(uint16_t nameLength)
{
  return nameLength + // Name V
         1 + 1 +      // MetaInfo
         1 + 5;       // Content TL
}

/** \brief Get required tailroom for EncodeData1 output mbuf, assuming max name length.
 */
static uint16_t
EncodeData1_GetTailroomMax()
{
  return EncodeData1_GetTailroom(NAME_MAX_LENGTH);
}

/** \brief Make a Data, step1.
 *  \param m output mbuf, must be empty and is the only segment, must have
 *           \p EncodeData1_GetHeadroom() in headroom and
 *           \p EncodeData1_GetTailroom(name) in tailroom;
 *           headroom for Ethernet and NDNLP headers shall be included if needed.
 *  \param name the Data name; this function will copy the name
 *  \param payload the payload; this function will chain them onto \p m , so they should be
 *                 indirect mbufs if shared
 */
void EncodeData1(struct rte_mbuf* m, LName name, struct rte_mbuf* payload);

static uint16_t
EncodeData2_GetHeadroom()
{
  return 0;
}

extern const uint16_t __EncodeData2_FakeSigLen;

static uint16_t
EncodeData2_GetTailroom()
{
  return __EncodeData2_FakeSigLen;
}

/** \brief Make a Data, step2.
 *  \param m signature mbuf, must be empty and is the only segment, must have
 *           \p EncodeData2_GetHeadroom() in headroom and
 *           \p EncodeData2_GetTailroom() in tailroom
 *  \param data1 'm' from \p EncodeData1
 *
 *  This function prepares a fake signature in \p m and chains it onto the Data.
 */
void EncodeData2(struct rte_mbuf* m, struct rte_mbuf* data1);

/** \brief Make a Data, step3.
 *  \param data2 'data1' from \p EncodeData2
 *
 *  This function prepends TLV-TYPE and TLV-LENGTH of Data element in the first segment.
 *  \p EncodeData1_GetHeadroom() has accounted for these octets.
 */
void EncodeData3(struct rte_mbuf* data2);

#endif // NDN_DPDK_NDN_DATA_H
