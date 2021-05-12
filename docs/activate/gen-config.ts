import type { TgConfig } from "@usnistgov/ndn-dpdk";
import stdout from "stdout-stream";

const cfg: TgConfig = {
  face: {
    scheme: "ether",
    local: "02:00:00:00:00:01",
    remote: "01:00:5e:00:17:aa",
    mtu: 8800,
    portConfig: {
      mtu: 9000,
    },
    vdevConfig: {
      xdp: {
        disabled: true,
      },
      afPacket: {
        disabled: true,
      },
    },
  },
  producer: {
    patterns: [
      {
        prefix: "/P/0",
        replies: [
          {
            payloadLen: 8000,
          },
        ],
      },
      {
        prefix: "/P/1",
        replies: [
          {
            weight: 5,
            suffix: "/S100",
            freshnessPeriod: "100ms",
            payloadLen: 100,
          },
          {
            weight: 5,
            suffix: "/S200",
            freshnessPeriod: "200ms",
            payloadLen: 200,
          },
          {
            weight: 2,
            nack: 100,
          },
          {
            weight: 1,
            timeout: true,
          },
        ],
      },
    ],
  },
  consumer: {
    patterns: [
      {
        weight: 10,
        prefix: "/Q/0",
      },
      {
        weight: 1,
        prefix: "/Q/1",
        canBePrefix: true,
        mustBeFresh: true,
      },
    ],
    interval: "1ms",
  },
};

stdout.write(JSON.stringify(cfg));
