import type { ActivateGenArgs } from "@usnistgov/ndn-dpdk";
import stdout from "stdout-stream";

const args: ActivateGenArgs = {
  eal: {
    pciDevices: [
      "0000:04:00.0",
    ],
  },
  mempool: {
    DIRECT: { capacity: 1048575, dataroom: 9128 },
    INDIRECT: { capacity: 1048575 },
  },
};

stdout.write(JSON.stringify(args));
