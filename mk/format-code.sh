#!/bin/bash
set -eo pipefail

git ls-files 'csrc/**/*.[hc]' | grep -v csrc/vendor | \
  xargs clang-format-8 -i -style=file

gofmt -l -w -s .

node_modules/.bin/xo --fix

git ls-files '*.yml' '*.yaml' | xargs yamllint

node_modules/.bin/markdownlint --ignore node_modules '**/*.md'
