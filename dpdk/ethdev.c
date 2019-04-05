#include "ethdev.h"

static struct ether_addr ethPcapDefaultMac = {
  .addr_bytes = { 0, 0, 0, 0x1, 0x2, 0x3 }
};

void
EthDev_GetMacAddr(uint16_t port, struct ether_addr* macaddr)
{
  rte_eth_macaddr_get(port, macaddr);

  if (memcmp(macaddr, &ethPcapDefaultMac, 6) == 0) {
    for (int i = 0; i < 6; ++i) {
      macaddr->addr_bytes[i] = lrand48();
    }
    macaddr->addr_bytes[0] &= 0xFE;
  }
}
