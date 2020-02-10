const path = require("path");
const jsRc = require("./eslintrc-js");

module.exports = {
  parser: "@typescript-eslint/parser",
  parserOptions: {
    project: path.resolve(__dirname, "..", "tsconfig.json"),
  },
  plugins: [
    "@typescript-eslint/eslint-plugin",
    "simple-import-sort",
  ],
  env: {
    node: true,
  },
  extends: [
    "eslint:recommended",
    "plugin:@typescript-eslint/eslint-recommended",
    "plugin:@typescript-eslint/recommended",
    "plugin:@typescript-eslint/recommended-requiring-type-checking",
  ],
  rules: {
    ...jsRc.rules,
    "@typescript-eslint/ban-ts-ignore": "off",
    "@typescript-eslint/explicit-function-return-type": "off",
    "@typescript-eslint/member-delimiter-style": ["error", { singleline: { delimiter: "comma" } }],
    "@typescript-eslint/no-empty-interface": ["error", { allowSingleExtends: true }],
    "@typescript-eslint/no-explicit-any": "off",
    "@typescript-eslint/no-inferrable-types": ["error", { ignoreParameters: true, ignoreProperties: true }],
    "@typescript-eslint/no-misused-promises": ["error", { checksVoidReturn: false }],
    "@typescript-eslint/no-namespace": "off",
    "@typescript-eslint/no-non-null-assertion": "off",
    "@typescript-eslint/no-unused-vars": ["warn", { args: "none" }],
    "@typescript-eslint/no-use-before-define": "off",
    "@typescript-eslint/require-await": "off",
    "@typescript-eslint/unbound-method": ["error", { ignoreStatic: true }],
    "constructor-super": "off",
    "no-inner-declarations": "off",
    "require-atomic-updates": "off",
  },
};
