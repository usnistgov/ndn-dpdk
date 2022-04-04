#!/bin/bash
set -euo pipefail
exec $GODEFCC $(pkg-config --cflags libdpdk) -DGODEF "$@"
