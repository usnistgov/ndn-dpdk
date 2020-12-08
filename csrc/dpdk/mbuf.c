#include "mbuf.h"

static_assert(sizeof(rte_mbuf_timestamp_t) == sizeof(TscTime), "");

int Mbuf_Timestamp_DynFieldOffset_ = -1;

bool
Mbuf_RegisterDynFields()
{
  int res = rte_mbuf_dyn_rx_timestamp_register(&Mbuf_Timestamp_DynFieldOffset_, NULL);
  return res == 0;
}
