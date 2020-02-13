import * as jayson from "jayson";
import * as yargs from "yargs";

import * as ethface from "../../iface/ethface/mod.js";
import * as iface from "../../iface/mod.js";
import * as facemgmt from "../../mgmt/facemgmt/mod.js";
import * as mgmt from "../../mgmt/mod.js";

const args = yargs
  .option("scheme", {
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

const loc = {
  Scheme: args.scheme,
  Port: args.port,
  Local: args.local,
  Remote: args.remote,
} as ethface.Locator;
if (args.vlan) {
  loc.Vlan = [args.vlan];
}

const mgmtClient = new mgmt.RpcClient(jayson.Client.tcp({port: 6345}));
mgmtClient.request<iface.Locator, facemgmt.BasicInfo>("Face.Create", loc)
.then((result: facemgmt.BasicInfo) => {
  process.stdout.write(result.Id.toString() + "\n");
  process.exit(0);
})
.catch((err) => {
  process.stderr.write(JSON.stringify(err) + "\n");
  process.exit(1);
});
