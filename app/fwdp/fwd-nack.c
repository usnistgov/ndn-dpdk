#include "fwd.h"

#include "../../core/logger.h"

INIT_ZF_LOG(FwFwd);

void
FwFwd_RxNack(FwFwd* fwd, Packet* npkt)
{
  struct rte_mbuf* pkt = Packet_ToMbuf(npkt);
  uint64_t token = Packet_GetLpL3Hdr(npkt)->pitToken;

  ZF_LOGD("nack-from=%" PRI_FaceId " npkt=%p up-token=%016" PRIx64, pkt->port,
          npkt, token);

  rte_pktmbuf_free(pkt);
}
