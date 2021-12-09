import type { ActivateGenArgs } from "@usnistgov/ndn-dpdk";
import stdout from "stdout-stream";

const args: ActivateGenArgs = {
  mempool: {
    DIRECT: { capacity: 1048575, dataroom: 9146 },
    INDIRECT: { capacity: 1048575 },
  },
};

stdout.write(JSON.stringify(args));
