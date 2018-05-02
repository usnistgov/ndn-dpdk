#!/bin/bash
R="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

COMMIT=$(git -C $R describe --match=NeVeRmAtCh --always --abbrev=40 --dirty)

(
  echo 'package version'
  echo 'import "time"'
  echo
  echo 'const COMMIT = "'$COMMIT'"'
  echo 'func GetBuildTime() time.Time {'
  echo 'return time.Unix('$(date +%s)',0)'
  echo '}'
) | gofmt > $R/version.go
