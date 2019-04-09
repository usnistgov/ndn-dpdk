import { ArgumentParser } from "argparse";
import * as fs from "fs";
import * as loglevel from "loglevel";
import * as loglevelPrefix from "loglevel-plugin-prefix";
import * as net from "net";

import { Transfer } from "./transfer";

interface IArgs {
  listener: string;
  verbosity: number;
}
const parser = new ArgumentParser({
  addHelp: true,
  description: "NFD emulator that allows application written for NFD to work with ndn-dpdk forwarder",
});
parser.addArgument("-l", { dest: "listener", defaultValue: "/tmp/nfdemu.sock", help: "Unix socket listener path" });
parser.addArgument("-v", { action: "count", dest: "verbosity", help: "increase verbosity" });
const args: IArgs = parser.parseArgs();

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

if (fs.existsSync(args.listener)) {
  fs.unlinkSync(args.listener);
}

const server = new net.Server();
server.on("connection", (socket: net.Socket) => { new Transfer(socket).begin(); });
server.listen(args.listener);
