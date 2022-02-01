import type { ActivateFwArgs } from "@usnistgov/ndn-dpdk";
import stdout from "stdout-stream";

const args: ActivateFwArgs = {
  eal: {
    cores: [6, 7, 8, 9, 24, 25, 26, 27],
  },
  mempool: {
    DIRECT: { capacity: 1048575, dataroom: 9146 },
    INDIRECT: { capacity: 1048575 },
  },
  fib: {
    capacity: 4095,
    startDepth: 8,
  },
  pcct: {
    pcctCapacity: 65535,
    csMemoryCapacity: 20000,
    csIndirectCapacity: 20000,
  },
};

stdout.write(JSON.stringify(args));
