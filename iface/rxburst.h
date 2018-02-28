#ifndef NDN_DPDK_IFACE_RXCB_H
#define NDN_DPDK_IFACE_RXCB_H

/// \file

#include "common.h"

/** \brief A burst of received packets.
 */
typedef struct FaceRxBurst
{
  uint16_t capacity;   ///< capacity of each L3 type
  uint16_t nInterests; ///< Interest count
  uint16_t nData;      ///< Data count
  uint16_t nNacks;     ///< Nack count
  Packet* npkt[0];
} FaceRxBurst;

/** \brief Allocate a FaceRxBurst of specified capacity.
 */
FaceRxBurst* FaceRxBurst_New(uint16_t capacity);

void FaceRxBurst_Close(FaceRxBurst* burst);

/** \brief Access the array of Interests.
 */
static Packet**
FaceRxBurst_ListInterests(FaceRxBurst* burst)
{
  return &burst->npkt[0];
}

/** \brief Access the array of Data.
 */
static Packet**
FaceRxBurst_ListData(FaceRxBurst* burst)
{
  return &burst->npkt[burst->capacity];
}

/** \brief Access the array of Nacks.
 */
static Packet**
FaceRxBurst_ListNacks(FaceRxBurst* burst)
{
  return &burst->npkt[burst->capacity + burst->capacity];
}

/** \brief Get i-th Interest.
 */
static Packet*
FaceRxBurst_GetInterest(FaceRxBurst* burst, uint16_t i)
{
  assert(i < burst->nInterests);
  return FaceRxBurst_ListInterests(burst)[i];
}

/** \brief Get i-th Data.
 */
static Packet*
FaceRxBurst_GetData(FaceRxBurst* burst, uint16_t i)
{
  assert(i < burst->nData);
  return FaceRxBurst_ListData(burst)[i];
}

/** \brief Get i-th Nack.
 */
static Packet*
FaceRxBurst_GetNack(FaceRxBurst* burst, uint16_t i)
{
  assert(i < burst->nNacks);
  return FaceRxBurst_ListNacks(burst)[i];
}

/** \brief Get a scratch space for receiving up to \p burst->capacity frames.
 *
 *  This scratch space overlaps with the space for Interests. It is safe to use
 *  \p FaceRxBurst_PutInterest as long as processing each frames adds at most
 *  one Interest.
 */
static struct rte_mbuf**
FaceRxBurst_GetScratch(FaceRxBurst* burst)
{
  return (struct rte_mbuf**)burst->npkt;
}

/** \brief Clear all packets.
 *  \note This does not deallocate packets.
 */
static void
FaceRxBurst_Clear(FaceRxBurst* burst)
{
  burst->nInterests = 0;
  burst->nData = 0;
  burst->nNacks = 0;
}

/** \brief Add an Interest.
 *  \pre burst->nInterests < burst->capacity
 */
static void
FaceRxBurst_PutInterest(FaceRxBurst* burst, Packet* npkt)
{
  assert(burst->nInterests < burst->capacity);
  assert(Packet_GetL3PktType(npkt) == L3PktType_Interest);
  FaceRxBurst_ListInterests(burst)[burst->nInterests++] = npkt;
}

/** \brief Add a Data.
 *  \pre burst->nData < burst->capacity
 */
static void
FaceRxBurst_PutData(FaceRxBurst* burst, Packet* npkt)
{
  assert(burst->nData < burst->capacity);
  assert(Packet_GetL3PktType(npkt) == L3PktType_Data);
  FaceRxBurst_ListData(burst)[burst->nData++] = npkt;
}

/** \brief Add a Nack.
 *  \pre burst->nNacks < burst->capacity
 */
static void
FaceRxBurst_PutNack(FaceRxBurst* burst, Packet* npkt)
{
  assert(burst->nNacks < burst->capacity);
  assert(Packet_GetL3PktType(npkt) == L3PktType_Nack);
  FaceRxBurst_ListNacks(burst)[burst->nNacks++] = npkt;
}

/** \brief Callback upon packet arrival.
 */
typedef void (*Face_RxCb)(Face* face, FaceRxBurst* burst, void* cbarg);

#endif // NDN_DPDK_IFACE_RXCB_H
