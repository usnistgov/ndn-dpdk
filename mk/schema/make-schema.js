import jsonStringify from "json-stable-stringify";

import { makeSchema } from "./tjs-schema.js";

const filename = process.argv[2];
const typ = process.argv[3];
const schema = makeSchema(filename, typ);

process.stdout.write(jsonStringify(schema, { space: 2 }));
