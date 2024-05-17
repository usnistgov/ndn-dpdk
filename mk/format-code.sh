#!/bin/bash
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")"/..
LANG=${1:-}
LANG=${LANG,,}

# C
if [[ -z $LANG ]] || [[ $LANG == c ]]; then
  git ls-files -- 'csrc/*.[hc]' 'bpf/*.[hc]' -x ':!:csrc/vendor' | xargs clang-format-15 -i -style=file
fi

# Go
if [[ -z $LANG ]] || [[ $LANG == go ]]; then
  source mk/goenv.sh
  gofmt -l -w -s .
  go mod tidy
  staticcheck ./...
fi

# bash
if [[ -z $LANG ]] || [[ $LANG == sh ]]; then
  git ls-files -- '*.sh' | xargs shfmt -l -w -s -i=2 -ci
fi

# TypeScript
if [[ -z $LANG ]] || [[ $LANG == ts ]]; then
  $(corepack pnpm bin)/xo-yoursunny --fix
fi

# YAML
if [[ -z $LANG ]] || [[ $LANG == yaml ]]; then
  git ls-files '*.yml' '*.yaml' '.clang-format' | xargs yamllint -c mk/yamllint.yaml
fi

# Markdown
if [[ -z $LANG ]] || [[ $LANG == md ]]; then
  git ls-files '*.md' | xargs $(corepack pnpm bin)/markdownlint
fi

# Docker
if ([[ -z $LANG ]] || [[ $LANG == docker ]]) && command -v docker &>/dev/null; then
  git ls-files -- Dockerfile '*/Dockerfile' | xargs docker run --rm -u $(id -u):$(id -g) -v $PWD:/mnt -w /mnt hadolint/hadolint hadolint -t error
fi
