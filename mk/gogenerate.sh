#!/bin/bash
set -e
set -o pipefail

PKG=$1
if [[ -z $PKG ]]; then
  PKG=./...
fi
go generate $PKG
