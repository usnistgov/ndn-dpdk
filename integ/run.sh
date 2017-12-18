#!/bin/bash
cd "$( dirname "${BASH_SOURCE[0]}" )"
NPASSES=0
NFAILS=0
for T in */*/; do
  T=${T:0:-1}
  if [[ -n $1 ]] && [[ $T != $1 ]]; then continue; fi
  EXECUTABLE=integ_$(echo $T | sed 's~/~_~')
  echo -e '\033[0;36m'EXEC $T'\033[0m'
  rm -f /tmp/$EXECUTABLE
  go build -o /tmp/$EXECUTABLE ndn-dpdk/integ/$T
  if sudo /tmp/$EXECUTABLE; then
    echo -e '\033[0;32m'PASS $T'\033[0m'
    NPASSES=$((NPASSES+1))
  else
    echo -e '\033[0;31m'FAIL $T'\033[0m'
    NFAILS=$((NFAILS+1))
  fi
done
echo SUMMARY: $NPASSES passed, $NFAILS failed