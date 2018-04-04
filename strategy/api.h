#ifndef NDN_DPDK_STRATEGY_API_H
#define NDN_DPDK_STRATEGY_API_H

/// \file

#include "api-struct.h"

typedef int64_t TscDuration;

/** \brief Set a timer to invoke strategy after a duration.
 *
 *  \c Program will be invoked again with \c SGEVT_TIMER after \p after.
 *  However, the timer would be cancelled if \c Program is invoked for any other event,
 *  a different timer is set, or the strategy choice has been changed.
 */
void SetTimer(SgCtx* ctx, TscDuration after);

/** \brief Forward an Interest to a nexthop.
 */
void ForwardInterest(SgCtx* ctx, FaceId nh);

/** \brief Return Nacks downstream and erase PIT entry.
 */
void ReturnNacks(SgCtx* ctx);

/** \brief The strategy program.
 *  \return status code, ignored by forwarding but appears in logs.
 *
 *  Every strategy must implement this function.
 */
uint64_t Program(SgCtx* ctx);

#endif // NDN_DPDK_STRATEGY_API_H
