#!/usr/bin/env node

import * as yargs from "yargs";

import * as ethface from "../../iface/ethface/mod";
import * as mgmt from "../../mgmt/mod";

const args = yargs
  .option("scheme", {
    choices: ["ether"],
    default: "ether",
    type: "string",
  })
  .option("port", {
    demandOption: true,
    type: "string",
  })
  .option("local", {
    type: "string",
  })
  .option("remote", {
    type: "string",
  })
  .option("vlan", {
    type: "number",
  }).parse();

const loc: ethface.Locator = {
  Scheme: "ether",
  Port: args.port,
  Local: args.local,
  Remote: args.remote,
};
if (args.vlan) {
  loc.Vlan = [args.vlan];
}

const mgmtClient = mgmt.RpcClient.create();
mgmtClient.request("Face", "Create", loc)
  .then((result) => {
    process.stdout.write(`${result.Id}\n`);
    process.exit(0);
  })
  .catch((err) => {
    process.stderr.write(`${JSON.stringify(err)}\n`);
    process.exit(1);
  });
