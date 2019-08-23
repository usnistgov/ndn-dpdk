#!/bin/bash
set -e

[[ -f kernel-headers.tgz ]] || tar -chzf kernel-headers.tgz /lib/modules/$(uname -r)
docker build -t ndn-dpdk . && \
rm kernel-headers.tgz
