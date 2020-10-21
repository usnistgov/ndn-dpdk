import jsonStringify = require("json-stable-stringify");
import * as path from "path";
import * as TJS from "typescript-json-schema";

const inputFile = process.argv[2];
const inputType = process.argv[3];

const tjsArgs: TJS.PartialArgs = {};
tjsArgs.validationKeywords = ["contentEncoding", "contentMediaType"];

const program = TJS.getProgramFromFiles([path.resolve(__dirname, "..", inputFile)]);
const schema = TJS.generateSchema(program, inputType, tjsArgs)!;

process.stdout.write(jsonStringify(schema, { space: 2 }));
