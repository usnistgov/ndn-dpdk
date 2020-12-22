#!/bin/bash
set -eo pipefail

PKG=$1
if [[ -z $PKG ]]; then
  PKG=./...
fi
go generate $PKG
