#include "fwd.h"

#include "../../core/logger.h"

INIT_ZF_LOG(FwNack);

void
FwFwd_RxNack(FwFwd* fwd, Packet* npkt)
{
  ZF_LOGD("%" PRIx8 " RxNack", fwd->id);
  rte_pktmbuf_free(Packet_ToMbuf(npkt));
}
