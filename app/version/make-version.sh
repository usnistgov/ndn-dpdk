#!/bin/bash
R="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

COMMIT=$(git -C $R describe --match=NeVeRmAtCh --always --abbrev=40 --dirty)

(
  echo 'package version'
  echo
  echo 'const COMMIT = "'$COMMIT'"'
) | gofmt -s > $R/version.go.new

if ! diff $R/version.go $R/version.go.new &>/dev/null; then
  mv $R/version.go.new $R/version.go
else
  rm $R/version.go.new
fi
