#include "gtpip-table.h"
#include "../core/logger.h"
#include "face.h"

N_LOG_INIT(GtpipTable);

bool
GtpipTable_ProcessDownlink(GtpipTable* table, struct rte_mbuf* m) {
  if (unlikely(m->data_len < RTE_ETHER_HDR_LEN + sizeof(struct rte_ipv4_hdr))) {
    return false;
  }
  const struct rte_ether_hdr* eth = rte_pktmbuf_mtod(m, const struct rte_ether_hdr*);
  const struct rte_ipv4_hdr* ip4 =
    rte_pktmbuf_mtod_offset(m, const struct rte_ipv4_hdr*, RTE_ETHER_HDR_LEN);
  if (eth->ether_type != rte_cpu_to_be_16(RTE_ETHER_TYPE_IPV4)) {
    return false;
  }
  rte_be32_t ueIP = ip4->dst_addr;

  void* hdata = NULL;
  int res = rte_hash_lookup_data(table->ipv4, &ueIP, &hdata);
  if (res < 0) {
    return false;
  }

  FaceID id = (FaceID)(uintptr_t)hdata;
  EthFacePriv* priv = Face_GetPriv(Face_Get(id));
  EthTxHdr* hdr = &priv->txHdr;
  EthTxHdr_Prepend(hdr, m, EthTxHdrFlagsGtpip);

  return true;
}

bool
GtpipTable_ProcessUplink(GtpipTable* table, struct rte_mbuf* pkt) {
  return false;
}
