import type { ActivateFwArgs, EalConfig } from "@usnistgov/ndn-dpdk";
import stdout from "stdout-stream";

const eal: EalConfig = {
  cores: [4, 5, 6, 7, 12, 13, 14, 15],
  pciDevices: [
    "0000:04:00.0",
  ],
  virtualDevices: [
    "net_af_packet1,iface=eth1",
  ],
};

const args: ActivateFwArgs = {
  eal,
  mempool: {
    DIRECT: { capacity: 1048575, dataroom: 9128 },
    INDIRECT: { capacity: 1048575 },
  },
  fib: {
    capacity: 4095,
    startDepth: 8,
  },
  pcct: {
    pcctCapacity: 65535,
    csDirectCapacity: 20000,
    csIndirectCapacity: 20000,
  },
};

stdout.write(JSON.stringify(args));
