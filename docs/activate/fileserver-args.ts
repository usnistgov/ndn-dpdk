import type { ActivateFileServerArgs } from "@usnistgov/ndn-dpdk";
import stdout from "stdout-stream";

const args: ActivateFileServerArgs = {
  eal: {
    coresPerNuma: { 0: 4, 1: 0 },
    memPerNuma: { 0: 2048, 1: 0 },
    filePrefix: "fileserver",
  },
  mempool: {
    DIRECT: { capacity: 65535, dataroom: 9128 },
    INDIRECT: { capacity: 65535 },
    PAYLOAD: { capacity: 65535, dataroom: 9128 },
  },
  face: {
    scheme: "memif",
    socketName: "/run/ndn/fileserver.sock",
    id: 0,
    role: "client",
    dataroom: 9000,
  },
  fileServer: {
    nThreads: 1,
    mounts: [
      { prefix: "/fileserver/usr-local-bin", path: "/usr/local/bin" },
      { prefix: "/fileserver/usr-local-lib", path: "/usr/local/lib" },
    ],
    segmentLen: 6 * 1024,
    uringCapacity: 4096,
  },
};

stdout.write(JSON.stringify(args));
