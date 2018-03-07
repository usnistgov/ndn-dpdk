#include "fwd.h"

#include "../../core/logger.h"

INIT_ZF_LOG(FwData);

void
FwFwd_RxData(FwFwd* fwd, Packet* npkt)
{
  ZF_LOGD("%" PRIu8 " RxData", fwd->id);
  rte_pktmbuf_free(Packet_ToMbuf(npkt));
}
