#!/bin/bash
set -e
set -o pipefail
MGMT=${MGMT:-tcp://127.0.0.1:6345}
TCPADDR=${MGMT#tcp*://}

if [[ $1 == 'help' ]]; then
  echo 'Endpoint: '$MGMT
  cat <<EOT
  Change endpoint with MGMT environment variable.
Subcommands:
  version [show]
    Show version.
  pingc [list]
    List ping clients.
  pingc start <I> <INTERVAL>
    Start i-th ping client.
  pingc stop <I>
    Stop i-th ping client.
  pingc counters <I>
    Show i-th ping client counters.
  fetch [list]
    List fetchers.
  fetch benchmark <I> <NAME> <INTERVAL> <COUNT>
    Run benchmark on i-th fetcher.
EOT
  exit 0
fi

HAS_JSONRPC=0
jsonrpc() {
  HAS_JSONRPC=1
  local METHOD=$1
  local PARAMS=$2
  if [[ -z $2 ]]; then PARAMS='{}'; fi
  jsonrpc2client -transport=tcp -tcp.addr=$TCPADDR $METHOD "$PARAMS"
}

if [[ $1 == 'version' ]]; then
  if [[ -z $2 ]] || [[ $2 == 'show' ]]; then
    jsonrpc Version.Version
  fi
elif [[ $1 == 'pingc' ]]; then
  if [[ -z $2 ]] || [[ $2 == 'list' ]]; then
    jsonrpc PingClient.List ''
  elif [[ $2 == 'start' ]]; then
    jsonrpc PingClient.Start '{"Index":'$3',"Interval":'$4',"ClearCounters":true}'
  elif [[ $2 == 'stop' ]]; then
    jsonrpc PingClient.Stop '{"Index":'$3'}'
  elif [[ $2 == 'counters' ]]; then
    jsonrpc PingClient.ReadCounters '{"Index":'$3'}'
  fi
elif [[ $1 == 'fetch' ]]; then
  if [[ -z $2 ]] || [[ $2 == 'list' ]]; then
    jsonrpc Fetch.List ''
  elif [[ $2 == 'benchmark' ]]; then
    jsonrpc Fetch.Benchmark '{"Index":'$3',"Templates":[{"Prefix":"'$4'"}],"Interval":'$5',"Count":'$6'}'
  fi
fi

if [[ $HAS_JSONRPC -eq 0 ]]; then
  echo 'Execute `'$0' help` to view usage.' >/dev/stderr
  exit 1
fi
