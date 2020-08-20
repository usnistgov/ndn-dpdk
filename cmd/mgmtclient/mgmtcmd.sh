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
  hrlog start <FILENAME>
    Start collecting high resolution logs.
  hrlog stop <FILENAME>
    Stop collecting high resolution logs.
  eth ports
    List Ethernet ports.
  eth faces <PORT>
    List Ethernet faces on port.
  eth stats <PORT>
    Read Ethernet port stats.
  eth reset-stats <PORT>
    Read and reset Ethernet port stats.
  ndt show
    Show NDT content.
  ndt counters
    Show NDT counters.
  ndt update <HASH> <VALUE>
    Update an NDT element by hash.
  ndt updaten <NAME> <VALUE>
    Update an NDT element by name.
  dpinfo [global]
    Show dataplane global information.
  dpinfo input <I>
  dpinfo fwd <I>
  dpinfo pit <I>
  dpinfo cs <I>
    Show dataplane i-th input/fwd/PIT/CS counters.
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
elif [[ $1 == 'hrlog' ]]; then
  if [[ $2 == 'start' ]]; then
    jsonrpc Hrlog.Start '{"Filename":"'$3'"}'
  elif [[ $2 == 'stop' ]]; then
    jsonrpc Hrlog.Stop '{"Filename":"'$3'"}'
  fi
elif [[ $1 == 'eth' ]]; then
  if [[ $2 == 'ports' ]]; then
    jsonrpc EthFace.ListPorts
  elif [[ $2 == 'faces' ]]; then
    jsonrpc EthFace.ListPortFaces '{"Port":"'$3'"}'
  elif [[ $2 == 'stats' ]]; then
    jsonrpc EthFace.ReadPortStats '{"Port":"'$3'","Reset":false}'
  elif [[ $2 == 'reset-stats' ]]; then
    jsonrpc EthFace.ReadPortStats '{"Port":"'$3'","Reset":true}'
  fi
elif [[ $1 == 'ndt' ]]; then
  if [[ $2 == 'show' ]]; then
    jsonrpc Ndt.ReadTable ''
  elif [[ $2 == 'counters' ]]; then
    jsonrpc Ndt.ReadCounters ''
  elif [[ $2 == 'update' ]]; then
    jsonrpc Ndt.Update '{"Hash":'$3',"Value":'$4'}'
  elif [[ $2 == 'updaten' ]]; then
    jsonrpc Ndt.Update '{"Name":"'$3'","Value":'$4'}'
  fi
elif [[ $1 == 'dpinfo' ]]; then
  if [[ -z $2 ]] || [[ $2 == 'global' ]]; then
    jsonrpc DpInfo.Global ''
  elif [[ $2 == 'input' ]] || [[ $2 == 'fwd' ]] || [[ $2 == 'pit' ]] || [[ $2 == 'cs' ]]; then
    jsonrpc DpInfo."${2^}" '{"Index":'$3'}'
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
