/** @typedef {import("xo").Options} XoOptions */

/** @type {import("@yoursunny/xo-config")} */
const { js, ts, web, preact, merge } = require("@yoursunny/xo-config");
const fs = require("node:fs");
const path = require("node:path");

/** @type {XoOptions} */
module.exports = {
  ...js,
  overrides: [
    {
      files: [
        "**/*.ts",
      ],
      ...merge(js, ts),
    },
    {
      files: [
        "js/types/**/*.ts",
      ],
      ...merge(js, ts, {
        rules: {
          "tsdoc/syntax": "off", // `@` tags are for ts-json-schema-generator
        },
      }),
    },
    {
      files: [
        "sample/benchmark/**/*.tsx",
        "sample/status/**/*.tsx",
      ],
      ...merge(js, ts, web, preact),
    },
  ],
  ignores: [
    "sample/activate",
    "sample/benchmark",
    "sample/status",
  ].filter((d) => !fs.statSync(path.resolve(__dirname, d, "node_modules"), { throwIfNoEntry: false })?.isDirectory()),
};
