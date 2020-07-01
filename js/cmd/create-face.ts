#!/usr/bin/env node

import * as yargs from "yargs";

import type { EthFaceLocator } from "../mod";
import { RpcClient } from "../mod";

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

const loc: EthFaceLocator = {
  Scheme: "ether",
  Port: args.port,
  Local: args.local,
  Remote: args.remote,
};
if (args.vlan) {
  loc.Vlan = [args.vlan];
}

const mgmtClient = RpcClient.create();
mgmtClient.request("Face", "Create", loc)
  .then((result) => {
    process.stdout.write(`${result.Id}\n`);
    process.exit(0);
  })
  .catch((err) => {
    process.stderr.write(`${JSON.stringify(err)}\n`);
    process.exit(1);
  });
