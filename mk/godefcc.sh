#!/bin/bash
$GODEFCC $(pkg-config --cflags libdpdk) "$@"
