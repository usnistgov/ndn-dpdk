#!/bin/bash
set -e
set -o pipefail

gofmt -l -w -s .
find -name '*.h' -o -name '*.c' \
  | grep -vE 'pcg_basic|siphash-20121104|uthash|zf_log' \
  | xargs clang-format-8 -i -style='{BasedOnStyle: Mozilla, ReflowComments: false}'
node_modules/.bin/xo --fix
find . -path ./node_modules -prune -o \( -name '*.yaml' -o -name '*.yml' \) -print | xargs yamllint
node_modules/.bin/markdownlint --ignore node_modules '**/*.md'
