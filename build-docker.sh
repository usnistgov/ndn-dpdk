#!/bin/bash
set -e

tar -chzf kernel-headers.tgz /lib/modules/$(uname -r)
docker build -t ndn-dpdk .
rm kernel-headers.tgz
