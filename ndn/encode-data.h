#ifndef NDN_DPDK_NDN_ENCODE_DATA_H
#define NDN_DPDK_NDN_ENCODE_DATA_H

/// \file

#include "name.h"

static uint16_t
EncodeData_GetHeadroom()
{
  return 1 + 5; // Data TL
}

extern const uint16_t __EncodeData_FakeSigLen;

static uint16_t
EncodeData_GetTailroom(uint16_t nameL, uint16_t contentL)
{
  return 1 + 3 + nameL +     // Name
         1 + 1 + 1 + 1 + 4 + // MetaInfo with FreshnessPeriod
         1 + 4 + contentL +  // Content
         __EncodeData_FakeSigLen;
}

/** \brief Get required tailroom for EncodeData output mbuf,
 *         assuming max name length and empty payload.
 */
static uint16_t
EncodeData_GetTailroomMax()
{
  return EncodeData_GetTailroom(NAME_MAX_LENGTH, 0);
}

void
__EncodeData(struct rte_mbuf* m,
             uint16_t namePrefixL,
             const uint8_t* namePrefixV,
             uint16_t nameSuffixL,
             const uint8_t* nameSuffixV,
             uint32_t freshnessPeriod,
             uint16_t contentL,
             const uint8_t* contentV);

/** \brief Encode a Data.
 *  \param m output mbuf, must be empty and is the only segment, must have
 *           \c EncodeData_GetHeadroom() in headroom and
 *           <tt>EncodeData_GetTailroom(namePrefix.length + nameSuffix.length,
 *           contentL)</tt> in tailroom; headroom for Ethernet and NDNLP
 *           headers may be included if needed.
 *  \param contentV the payload, will be copied.
 */
static void
EncodeData(struct rte_mbuf* m,
           LName namePrefix,
           LName nameSuffix,
           uint32_t freshnessPeriod,
           uint16_t contentL,
           const uint8_t* contentV)
{
  __EncodeData(m,
               namePrefix.length,
               namePrefix.value,
               nameSuffix.length,
               nameSuffix.value,
               freshnessPeriod,
               contentL,
               contentV);
}

#endif // NDN_DPDK_NDN_ENCODE_DATA_H
