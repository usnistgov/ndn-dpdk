#!/usr/bin/env node

import * as yargs from "yargs";

import { RpcClient } from "../mod";
import type { FaceLocator, SocketFaceLocator } from "../types/mod";

const args = yargs
  .option("scheme", {
    type: "string",
    choices: ["ether", "unix", "udp", "tcp"],
    default: "ether",
  })
  .option("port", {
    type: "string",
    desc: "local port name (either port or local must be specified for 'ether' scheme)",
  })
  .option("local", {
    type: "string",
    desc: "local MAC address or socket endpoint",
  })
  .option("remote", {
    type: "string",
    default: "01:00:5e:00:17:aa",
    desc: "remote MAC address or socket endpoint",
  })
  .option("vlan", {
    type: "number",
  })
  .check(({ scheme, local }) => {
    if (scheme === "ether" && !local) {
      throw new Error("--local is required for 'ether' scheme");
    }
    return true;
  })
  .parse();

async function main() {
  const mgmtClient = RpcClient.create();

  let loc: FaceLocator;
  if (args.scheme === "ether") {
    loc = {
      scheme: "ether",
      port: args.port,
      local: args.local!,
      remote: args.remote,
      vlan: args.vlan,
    };
  } else {
    loc = {
      scheme: args.scheme as SocketFaceLocator["scheme"],
      local: args.local,
      remote: args.remote,
    };
  }

  const created = await mgmtClient.request("Face", "Create", loc);
  process.stdout.write(`${created.Id}\n`);
  mgmtClient.close();
}

main()
  .catch((err) => {
    process.stderr.write(`${err}\n`);
    process.exit(1);
  });
