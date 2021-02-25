#!/bin/bash
set -eo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")"/..

# C
git ls-files -- 'csrc/**/*.[hc]' 'strategy/*.[hc]' -x ':!:csrc/vendor' | xargs clang-format-8 -i -style=file

# Go
go mod tidy
gofmt -l -w -s .

# TypeScript
node_modules/.bin/xo --fix

# YAML
git ls-files '*.yml' '*.yaml' | xargs yamllint

# Markdown
node_modules/.bin/markdownlint --ignore node_modules '**/*.md'

# Docker
node_modules/.bin/dockerfilelint Dockerfile
