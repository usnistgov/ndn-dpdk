#include "fwd.h"

#include "../../core/logger.h"

INIT_ZF_LOG(FwInterest);

void
FwFwd_RxInterest(FwFwd* fwd, Packet* npkt)
{
  ZF_LOGD("%" PRIu8 " RxInterest", fwd->id);
  rte_pktmbuf_free(Packet_ToMbuf(npkt));
}
