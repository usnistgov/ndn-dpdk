#!/bin/bash
set -euo pipefail
source "$( dirname "${BASH_SOURCE[0]}" )"/cflags.sh
export CGO_CFLAGS_ALLOW='.*'
export GOAMD64=${GOAMD64:-v2}
exec go "$@"
