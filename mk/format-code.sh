#!/bin/bash
gofmt -l -w -s .
find -name '*.h' -o -name '*.c' \
  | grep -vE 'pcg_basic|siphash-20121104|uthash|zf_log' \
  | xargs clang-format-6.0 -i -style='{BasedOnStyle: Mozilla, ReflowComments: false}'
node_modules/.bin/eslint --ignore-path .gitignore -c mk/eslintrc-ts.js --fix **/*.ts
node_modules/.bin/eslint --ignore-path .gitignore -c mk/eslintrc-js.js --fix **/*.js
find . -path ./node_modules -prune -o \( -name '*.yaml' -o -name '*.yml' \) -print | xargs yamllint
node_modules/.bin/markdownlint --ignore node_modules '**/*.md'
