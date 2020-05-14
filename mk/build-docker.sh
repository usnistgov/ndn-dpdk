#!/bin/bash
set -e
set -o pipefail
docker build -t ndn-dpdk .
