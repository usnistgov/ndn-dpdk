#ifndef NDN_DPDK_NDN_TLV_ENCODER_H
#define NDN_DPDK_NDN_TLV_ENCODER_H

/** \file
 *
 *  \par Common return values of decoding functions:
 *  \retval NdnError_OK successful; encoder is advanced past end of encoded item.
 *  \retval NdnError_Incomplete reaching output boundary before encoding finishes.
 */

#include "common.h"

/** \brief TLV encoder.
 */
typedef struct TlvEncoder
{
} TlvEncoder;

/** \brief Cast mbuf as TlvEncoder.
 *
 *  The mbuf must be the only segment and must be empty.
 */
static inline TlvEncoder*
MakeTlvEncoder(struct rte_mbuf* m)
{
  assert(m->nb_segs == 1 && m->pkt_len == 0 && m->data_len == 0);
  return (TlvEncoder*)(void*)m;
}

static inline uint8_t*
TlvEncoder_Append(TlvEncoder* en, uint16_t len)
{
  struct rte_mbuf* m = (struct rte_mbuf*)en;
  if (unlikely(len > rte_pktmbuf_tailroom(m))) {
    return NULL;
  }
  uint16_t off = m->data_len;
  m->pkt_len = m->data_len = off + len;
  return rte_pktmbuf_mtod_offset(m, uint8_t*, off);
}

static inline uint8_t*
TlvEncoder_Prepend(TlvEncoder* en, uint16_t len)
{
  struct rte_mbuf* m = (struct rte_mbuf*)en;
  return rte_pktmbuf_prepend(m, len);
}

/** \brief Compute size of a TLV-TYPE or TLV-LENGTH number.
 */
static inline int
SizeofVarNum(uint64_t n)
{
  return n <= UINT16_MAX ? (n < 253 ? 1 : 3) : (n <= UINT32_MAX ? 5 : 9);
}

void __EncodeVarNum_32or64(uint8_t* room, uint64_t n);

/** \brief Encode a TLV-TYPE or TLV-LENGTH number.
 *  \param[out] room output buffer, must have \p SizeofVarNum(n) octets
 *  \param n the number
 */
static inline void
EncodeVarNum(uint8_t* room, uint64_t n)
{
  if (unlikely(n >= UINT16_MAX)) {
    __EncodeVarNum_32or64(room, n);
    return;
  }

  if (n < 253) {
    room[0] = (uint8_t)n;
  } else {
    room[0] = 253;
    room[1] = (uint8_t)(n >> 8);
    room[2] = (uint8_t)n;
  }
}

/** \brief Append a TLV-TYPE or TLV-LENGTH number.
 */
static inline NdnError
AppendVarNum(TlvEncoder* en, uint64_t n)
{
  uint8_t* room = TlvEncoder_Append(en, SizeofVarNum(n));
  if (unlikely(room == NULL)) {
    return NdnError_Incomplete;
  }

  EncodeVarNum(room, n);
  return NdnError_OK;
}

/** \brief Prepend a TLV-TYPE or TLV-LENGTH number.
 */
static inline NdnError
PrependVarNum(TlvEncoder* en, uint64_t n)
{
  uint8_t* room = TlvEncoder_Prepend(en, SizeofVarNum(n));
  if (unlikely(room == NULL)) {
    return NdnError_Incomplete;
  }

  EncodeVarNum(room, n);
  return NdnError_OK;
}

#endif // NDN_DPDK_NDN_TLV_ENCODER_H
