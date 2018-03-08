#include "fwd.h"

#include "../../core/logger.h"

INIT_ZF_LOG(FwNack);

void
FwFwd_RxNack(FwFwd* fwd, Packet* npkt)
{
  ZF_LOGD("%" PRIu8 " %p RxNack", fwd->id, npkt);
  rte_pktmbuf_free(Packet_ToMbuf(npkt));
}
