#ifndef NDN_DPDK_NDN_TLV_ENCODER_H
#define NDN_DPDK_NDN_TLV_ENCODER_H

/** \file
 *
 *  \par Common return values of encoding functions:
 *  \retval NdnError_OK successful; encoder is advanced past end of encoded item.
 *  \retval NdnError_Incomplete reaching output boundary before encoding finishes.
 */

#include "tlv-varnum.h"

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

static inline TlvEncoder*
MakeTlvEncoder_Unchecked(struct rte_mbuf* m)
{
  return (TlvEncoder*)(void*)m;
}

static inline struct rte_mbuf*
TlvEncoder_AsMbuf(TlvEncoder* en)
{
  return (struct rte_mbuf*)en;
}

static inline uint8_t*
TlvEncoder_Append(TlvEncoder* en, uint16_t len)
{
  struct rte_mbuf* m = TlvEncoder_AsMbuf(en);
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
  struct rte_mbuf* m = TlvEncoder_AsMbuf(en);
  return (uint8_t*)rte_pktmbuf_prepend(m, len);
}

/** \brief Append a TLV-TYPE or TLV-LENGTH number.
 */
static inline NdnError
AppendVarNum(TlvEncoder* en, uint32_t n)
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
PrependVarNum(TlvEncoder* en, uint32_t n)
{
  uint8_t* room = TlvEncoder_Prepend(en, SizeofVarNum(n));
  if (unlikely(room == NULL)) {
    return NdnError_Incomplete;
  }

  EncodeVarNum(room, n);
  return NdnError_OK;
}

#endif // NDN_DPDK_NDN_TLV_ENCODER_H
