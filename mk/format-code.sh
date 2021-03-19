#!/bin/bash
set -eo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")"/..

# C
git ls-files -- 'csrc/**/*.[hc]' 'strategy/*.[hc]' -x ':!:csrc/vendor' | xargs clang-format-8 -i -style=file

# Go
gofmt -l -w -s .
go mod tidy
staticcheck ./...

# TypeScript
node_modules/.bin/xo --fix

# YAML
git ls-files '*.yml' '*.yaml' | xargs yamllint

# Markdown
git ls-files '*.md' | xargs node_modules/.bin/markdownlint

# Docker
node_modules/.bin/dockerfilelint Dockerfile
