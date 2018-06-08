#!/bin/bash
set -e

make clean
mkdir -p build
tar -chf build/kernel-headers.tar /lib/modules/$(uname -r)
docker build -t ndn-dpdk .
