import { ArgumentParser } from "argparse";
import * as cluster from "cluster";
import * as fs from "fs";
import * as loglevel from "loglevel";
import * as loglevelPrefix from "loglevel-plugin-prefix";
import * as net from "net";
import * as os from "os";

import { Transfer } from "./transfer";

interface IArgs {
  listener: string;
  verbosity: number;
  nWorkers: number;
}
const parser = new ArgumentParser({
  addHelp: true,
  description: "NFD emulator that allows application written for NFD to work with ndn-dpdk forwarder",
});
parser.addArgument("-l", { defaultValue: "/tmp/nfdemu.sock",
                           dest: "listener",
                           help: "Unix socket listener path" });
parser.addArgument("-v", { action: "count",
                           dest: "verbosity",
                           help: "increase verbosity" });
parser.addArgument("-w", { defaultValue: Math.max(os.cpus().length, 4),
                           dest: "nWorkers",
                           help: "number of worker processes",
                           type: "int" });
const args: IArgs = parser.parseArgs();
args.nWorkers = Math.max(1, args.nWorkers);

if (args.verbosity >= 2) {
  loglevel.setLevel(loglevel.levels.DEBUG, false);
} else if (args.verbosity >= 1) {
  loglevel.setLevel(loglevel.levels.INFO, false);
} else {
  loglevel.setLevel(loglevel.levels.WARN, false);
}
loglevelPrefix.reg(loglevel);
loglevelPrefix.apply(loglevel, {
  template: "%n [%t]",
});

if (cluster.isMaster) {
  if (fs.existsSync(args.listener)) {
    fs.unlinkSync(args.listener);
  }

  for (let i = 0; i < args.nWorkers; ++i) {
    cluster.fork();
  }
} else {
  const server = new net.Server();
  server.on("connection", (socket: net.Socket) => { new Transfer(socket).begin(); });
  server.listen(args.listener);
}
