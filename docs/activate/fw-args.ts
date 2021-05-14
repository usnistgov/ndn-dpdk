import type { ActivateFwArgs, EalConfig } from "@usnistgov/ndn-dpdk";
import stdout from "stdout-stream";

const eal: EalConfig = {
  cores: [6, 7, 8, 9, 24, 25, 26, 27],
  pciDevices: [
    "0000:04:00.0",
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
