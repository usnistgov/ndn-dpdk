import { ArgumentParser } from "argparse";
import Debug = require("debug");
import getStdin = require("get-stdin");
import * as yaml from "js-yaml";
import * as objectPath from "object-path";

const debug = Debug("yamledit");

const parser = new ArgumentParser({
  addHelp: true,
  description: "YAML configuration editor",
});
parser.addArgument("-a", { action: "storeTrue", dest: "append", help: "append to list" });
parser.addArgument("-d", { action: "storeTrue", dest: "delete", help: "delete key" });
parser.addArgument("-j", { action: "storeTrue", dest: "isYaml", help: "value is JSON or YAML" });
parser.addArgument("-n", { action: "storeTrue", dest: "isNumber", help: "value is number" });
parser.addArgument("key", { help: "key path" });
parser.addArgument("value", { help: "new value", nargs: "?" });
const args = parser.parseArgs();

getStdin()
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
  process.stdout.write(yaml.safeDump(doc));
})
.catch(debug);
