import type { ActivateGenArgs, EalConfig } from "@usnistgov/ndn-dpdk";
import stdout from "stdout-stream";

const eal: EalConfig = {
  cores: [4, 5, 6, 7, 12, 13, 14, 15],
  pciDevices: [
    "0000:04:00.0",
  ],
};

const args: ActivateGenArgs = {
  eal,
  mempool: {
    DIRECT: { capacity: 1048575, dataroom: 9128 },
    INDIRECT: { capacity: 1048575 },
  },
  tasks: [
    {
      face: {
        scheme: "ether",
        local: "02:00:00:00:00:01",
        remote: "01:00:5e:00:17:aa",
      },
      consumer: {
        patterns: [{
          prefix: "/P/0",
        }],
        interval: "1ms",
      },
    },
    {
      face: {
        scheme: "ether",
        local: "02:00:00:00:00:02",
        remote: "01:00:5e:00:17:aa",
      },
      producer: {
        patterns: [{
          prefix: "/P/0",
          replies: [
            {
              payloadLen: 8000,
            },
          ],
        }],
      },
    },
  ],
};

stdout.write(JSON.stringify(args));
