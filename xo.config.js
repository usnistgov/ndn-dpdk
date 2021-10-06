/** @typedef {import("xo").Options} XoOptions */

/** @type {import("@yoursunny/xo-config")} */
const { js, ts, web, preact, merge } = require("@yoursunny/xo-config");

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
        "docs/benchmark/**/*.tsx",
      ],
      ...merge(js, ts, web, preact),
    },
  ],
};
