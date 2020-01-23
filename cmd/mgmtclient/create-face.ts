import { ArgumentParser } from "argparse";
import * as jayson from "jayson";

import * as ethface from "../../iface/ethface/mod.js";
import * as iface from "../../iface/mod.js";
import * as facemgmt from "../../mgmt/facemgmt/mod.js";
import * as mgmt from "../../mgmt/mod.js";

const parser = new ArgumentParser({
  addHelp: true,
  description: "Create a face",
});
parser.addArgument("--scheme", { required: true });
parser.addArgument("--port", { required: false });
parser.addArgument("--local", { required: false });
parser.addArgument("--remote", { required: true });
const args = parser.parseArgs();

const mgmtClient = new mgmt.RpcClient(jayson.Client.tcp({port: 6345}));
mgmtClient.request<iface.Locator, facemgmt.BasicInfo>("Face.Create",
  {
    Scheme: args.scheme,
    Port: args.port,
    Local: args.local,
    Remote: args.remote,
  } as ethface.Locator)
.then((result: facemgmt.BasicInfo) => {
  process.stdout.write(result.Id.toString() + "\n");
  process.exit(0);
})
.catch((err) => {
  process.stderr.write(JSON.stringify(err) + "\n");
  process.exit(1);
});
