#!/bin/bash
set -eo pipefail
NFD_CS_CAP=${NFD_CS_CAP:-65536}
NFD_ENABLE_ETHER=${NFD_ENABLE_ETHER:-0}
NFD_ENABLE_UDP=${NFD_ENABLE_UDP:-0}
export HOME=/var/lib/ndn/nfd

if ! ndnsec get-default &>/dev/null; then
  ndnsec key-gen /localhost/operator >/dev/null
fi

mkdir -p /etc/ndn/certs
ndnsec cert-dump -i $(ndnsec get-default) > /etc/ndn/certs/localhost.ndncert

cp /nfd.conf.sample /etc/ndn/nfd.conf
infoedit -f /etc/ndn/nfd.conf -s general.user -v ndn
infoedit -f /etc/ndn/nfd.conf -s general.group -v ndn
infoedit -f /etc/ndn/nfd.conf -s tables.cs_max_packets -v $NFD_CS_CAP
infoedit -f /etc/ndn/nfd.conf -s face_system.unix.path -v /run/ndn/nfd.sock
infoedit -f /etc/ndn/nfd.conf -d face_system.tcp
infoedit -f /etc/ndn/nfd.conf -d face_system.websocket
if [[ $NFD_ENABLE_UDP -eq 1 ]]; then
  infoedit -f /etc/ndn/nfd.conf -s face_system.udp.listen -v no
  infoedit -f /etc/ndn/nfd.conf -s face_system.udp.mcast -v no
else
  infoedit -f /etc/ndn/nfd.conf -d face_system.udp
fi
if [[ $NFD_ENABLE_ETHER -eq 1 ]]; then
  infoedit -f /etc/ndn/nfd.conf -s face_system.ether.listen -v no
  infoedit -f /etc/ndn/nfd.conf -s face_system.ether.mcast -v no
else
  infoedit -f /etc/ndn/nfd.conf -d face_system.ether
fi
infoedit -f /etc/ndn/nfd.conf -d rib.auto_prefix_propagate

chown -R ndn:ndn /var/lib/ndn/nfd
/connect.sh &
exec /usr/bin/nfd