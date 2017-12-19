#ifndef NDN_DPDK_NDN_NACK_PKT_H
#define NDN_DPDK_NDN_NACK_PKT_H

/// \file

/** \brief Indicate the Nack reason.
 */
typedef enum NackReason {
  NackReason_None = 0, ///< packet is not a Nack
  NackReason_Congestion = 50,
  NackReason_Duplicate = 100,
  NackReason_NoRoute = 150,
  NackReason_Unspecified = 255 ///< reason unspecified
} NackReason;

#endif // NDN_DPDK_NDN_NACK_PKT_H