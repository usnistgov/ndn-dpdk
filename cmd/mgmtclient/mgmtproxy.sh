#!/bin/bash
if [[ -z $1 ]]; then
  echo 'Usage: sudo mgmtproxy.sh start|stop [-v]' >/dev/stderr
  exit 2
fi

if [[ $1 == 'start' ]]; then
  socat $2 TCP-LISTEN:6345,reuseaddr,fork,bind=127.0.0.1 \
           UNIX-CONNECT:/var/run/ndn-dpdk-mgmt.sock &
  echo $! > /var/run/ndn-dpdk-mgmt-proxy.pid
fi

if [[ $1 == 'stop' ]]; then
  kill $(cat /var/run/ndn-dpdk-mgmt-proxy.pid)
fi
