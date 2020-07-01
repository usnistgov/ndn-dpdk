import jsonStringify = require("json-stable-stringify");
import * as path from "path";
import * as TJS from "typescript-json-schema";

import type { JrgenSpecSchema } from "../vendor/jrgen-spec-schema";

const tjsArgs: TJS.PartialArgs = {};
tjsArgs.validationKeywords = ["contentEncoding", "contentMediaType"];

const program = TJS.getProgramFromFiles([path.resolve(__dirname, "..", "types", "mgmt", "mgmt.ts")]);
const schema = TJS.generateSchema(program, "Mgmt", tjsArgs)!;

const spec: JrgenSpecSchema = {
  $schema: "https://unpkg.com/jrgen@2.0.0/jrgen-spec.schema.json",
  definitions: {},
  info: {
    title: "NdnDpdkMgmt",
    version: "0.0.0",
  },
  jrgen: "1.1",
  jsonrpc: "2.0",
  methods: {},
};

const mgmtModules: { [k: string]: string } = {};
for (const [propName, propSchema] of Object.entries(schema.properties!)) {
  const typeName = (propSchema as TJS.Definition).$ref!.replace("#/definitions/", "");
  mgmtModules[typeName] = propName;
}

for (const [typeName, typeSchema] of Object.entries(schema.definitions!)) {
  if (mgmtModules[typeName]) {
    const methodPrefix = mgmtModules[typeName];
    for (const [methodSuffix, methodSchema] of Object.entries((typeSchema as TJS.Definition).properties!)) {
      spec.methods[`${methodPrefix}.${methodSuffix}`] = {
        params: (methodSchema as TJS.Definition).properties!.args,
        result: (methodSchema as TJS.Definition).properties!.reply,
        summary: "",
      };
    }
  } else {
    spec.definitions![typeName] = typeSchema;
  }
}

process.stdout.write(jsonStringify(spec, { space: 2 }));
