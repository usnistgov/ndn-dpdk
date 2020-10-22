import * as TJS from "typescript-json-schema";

/**
 * @param {string} filename
 * @param {string} typ
 * @returns {TJS.Definition}
 */
export function makeSchema(filename, typ) {
  /** @type {TJS.PartialArgs} */
  const args = {};
  args.noExtraProps = true;
  args.required = true;
  args.validationKeywords = ["contentEncoding", "contentMediaType"];

  const program = TJS.getProgramFromFiles([filename]);
  return TJS.generateSchema(program, typ, args);
}
