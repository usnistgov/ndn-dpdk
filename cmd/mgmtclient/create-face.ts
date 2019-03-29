import { ArgumentParser } from "argparse";
import * as jayson from "jayson";

interface IFaceMgmtBasicInfo {
  Id: number;
  LocalUri: string;
  RemoteUri: string;
}

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
  ],
  (err, error, result: ReadonlyArray<IFaceMgmtBasicInfo>) => {
    if (err || error) {
      process.stderr.write((err || error).toString() + "\n");
      process.exit(1);
      return;
    }
    process.stdout.write(result[0].Id.toString() + "\n");
    process.exit(0);
  });
