import jsonStringify from "json-stable-stringify";
import * as TSJ from "ts-json-schema-generator";

const filename = process.argv[2];
const typ = process.argv[3];
const schema = TSJ.createGenerator({
  path: filename,
  type: typ,
}).createSchema(typ);

process.stdout.write(jsonStringify(schema, { space: 2 }));
