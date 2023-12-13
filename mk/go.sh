#!/bin/bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")"/cflags.sh
source "$(dirname "${BASH_SOURCE[0]}")"/goenv.sh
exec go "$@"
