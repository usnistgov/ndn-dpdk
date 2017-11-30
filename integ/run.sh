#!/bin/bash
cd "$( dirname "${BASH_SOURCE[0]}" )"
NPASSES=0
NFAILS=0
for T in */test-*/; do
  T=${T:0:-1}
  echo -e '\033[0;36m'EXEC $T'\033[0m'
  rm -f /tmp/$T
  go build -o /tmp/$T ndn-traffic-dpdk/integ/$T
  if sudo /tmp/$T; then
    echo -e '\033[0;32m'PASS $T'\033[0m'
    NPASSES=$((NPASSES+1))
  else
    echo -e '\033[0;31m'FAIL $T'\033[0m'
    NFAILS=$((NFAILS+1))
  fi
done
echo SUMMARY: $NPASSES passed, $NFAILS failed