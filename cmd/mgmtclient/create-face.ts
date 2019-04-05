import { ArgumentParser } from "argparse";
import * as jayson from "jayson";

import * as mgmt from "../../docs/mgmttypes";

const parser = new ArgumentParser({
  addHelp: true,
  description: "Create a face",
});
parser.addArgument("remote", { help: "remote FaceUri" });
parser.addArgument("local", { help: "local FaceUri" });
const args = parser.parseArgs();

const mgmtClient = jayson.Client.tcp({port: 6345});
mgmtClient.request("Face.Create",
  [
    {
      LocalUri: args.local,
      RemoteUri: args.remote,
    },
  ] as mgmt.facemgmt.CreateArg,
  (err, error, result: mgmt.facemgmt.CreateRes) => {
    if (err || error) {
      process.stderr.write((err || error).toString() + "\n");
      process.exit(1);
      return;
    }
    process.stdout.write(result[0].Id.toString() + "\n");
    process.exit(0);
  });
