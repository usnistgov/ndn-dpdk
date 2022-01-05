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
        "sample/benchmark/**/*.tsx",
      ],
      ...merge(js, ts, web, preact),
    },
  ],
  ignores: [
    "sample/activate",
    "sample/benchmark",
  ].filter((d) => !fs.statSync(path.resolve(__dirname, d, "node_modules"), { throwIfNoEntry: false })?.isDirectory()),
};
