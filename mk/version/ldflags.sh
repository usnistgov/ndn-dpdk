#!/bin/bash
set -eo pipefail
git show -s --format='-X github.com/usnistgov/ndn-dpdk/mk/version.commit=%H -X github.com/usnistgov/ndn-dpdk/mk/version.date=%ct'
git diff --quiet HEAD || echo '-X github.com/usnistgov/ndn-dpdk/mk/version.dirty=dirty'
