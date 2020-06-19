#!/bin/bash
set -e
set -o pipefail

ENUMTYPES=NdnError,L2PktType,L3PktType,DataSatisfyResult

stringer -type=$ENUMTYPES -output=enum_string.go

go run ../mk/enumgen/ -type=$ENUMTYPES -guard=NDN_DPDK_NDN_ENUM_H -out=../csrc/ndn/enum.h .
