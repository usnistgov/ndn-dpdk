import jsonStringify from "json-stable-stringify";

import { makeSchema } from "./tjs-schema.js";

const filename = process.argv[2];
const typ = process.argv[3];
const schema = makeSchema(filename, typ);

/** @type {import("./jrgen-spec-schema").JrgenSpecSchema} */
const spec = {
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

/** @type {Record<string, string>} */
const mgmtModules = {};
for (const [propName, propSchema] of Object.entries(schema.properties)) {
  const typeName = propSchema.$ref.replace("#/definitions/", "");
  mgmtModules[typeName] = propName;
}

for (const [typeName, typeSchema] of Object.entries(schema.definitions)) {
  if (mgmtModules[typeName]) {
    const methodPrefix = mgmtModules[typeName];
    for (const [methodSuffix, methodSchema] of Object.entries(typeSchema.properties)) {
      spec.methods[`${methodPrefix}.${methodSuffix}`] = {
        params: methodSchema.properties.args,
        result: methodSchema.properties.reply,
        summary: "",
      };
    }
  } else {
    spec.definitions[typeName] = typeSchema;
  }
}

process.stdout.write(jsonStringify(spec, { space: 2 }));
