import { ArgumentParser } from "argparse";
import Debug = require("debug");
import * as fs from "fs";
import getStream from "get-stream";
import * as yaml from "js-yaml";
import * as objectPath from "object-path";

const debug = Debug("yamledit");

interface IArgs {
  filename?: string;
  append?: boolean;
  delete?: boolean;
  isYaml?: boolean;
  isNumber?: boolean;
  key: string;
  value?: string;
}
const parser = new ArgumentParser({
  addHelp: true,
  description: "YAML configuration editor",
});
parser.addArgument("-f", { dest: "filename", help: "edit file in-place instead of stdin-stdout" });
parser.addArgument("-a", { action: "storeTrue", dest: "append", help: "append to list" });
parser.addArgument("-d", { action: "storeTrue", dest: "delete", help: "delete key" });
parser.addArgument("-j", { action: "storeTrue", dest: "isYaml", help: "value is JSON or YAML" });
parser.addArgument("-n", { action: "storeTrue", dest: "isNumber", help: "value is number" });
parser.addArgument("key", { help: "key path" });
parser.addArgument("value", { help: "new value", nargs: "?" });
const args: IArgs = parser.parseArgs();

getStream(args.filename ? fs.createReadStream(args.filename) : process.stdin)
.then((str) => yaml.safeLoad(str))
.then((doc) => {
  if (args.delete) {
    objectPath.del(doc, args.key);
    return doc;
  }

  let value: any = args.value;
  if (args.isNumber) {
    value = Number(value);
  } else if (args.isYaml) {
    value = yaml.safeLoad(value);
  }

  if (args.append) {
    objectPath.push(doc, args.key, value);
  } else {
    objectPath.set(doc, args.key, value);
  }

  return doc;
})
.then((doc) => {
  const output = yaml.safeDump(doc);
  if (args.filename) {
    fs.writeFileSync(args.filename, output);
  } else {
    process.stdout.write(output);
  }
})
.catch(debug);
