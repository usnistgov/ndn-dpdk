import * as jayson from "jayson";

import * as mgmt from "../../mgmt";

import * as msi from "./msi";
import { NdnpingTrafficGen } from "./trafficgen";

(async function main() {
  const rpcClient = new mgmt.RpcClient(jayson.Client.tcp({port: 6345}));
  const gen = await NdnpingTrafficGen.create(rpcClient);
  const res = await msi.measure(gen);
  process.stdout.write(JSON.stringify(res) + "\n");
})()
.catch((err) => {
  process.stderr.write(JSON.stringify(err) + "\n");
  process.exit(1);
});
