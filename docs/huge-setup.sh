#!/bin/bash
set -eo pipefail

SPDK_PATH=$HOME/code/spdk-20.10
HUGE2M_NPAGES=0
HUGE1G_NPAGES=8

if [[ -x "${SPDK_PATH}/scripts/setup.sh" ]]; then
  NRHUGE=0 eval "${SPDK_PATH}/scripts/setup.sh"
fi
[[ -f /mnt/huge ]] && umount /mnt/huge

if [[ $HUGE2M_NPAGES -gt 0 ]]; then
  if ! mount | grep /mnt/huge2M; then
    mkdir -p /mnt/huge2M
    mount -t hugetlbfs nodev /mnt/huge2M -o pagesize=2M
  fi
  echo $HUGE2M_NPAGES | tee /sys/devices/system/node/node*/hugepages/hugepages-2048kB/nr_hugepages
fi

if [[ $HUGE1G_NPAGES -gt 0 ]]; then
  if ! mount | grep /mnt/huge1G; then
    mkdir -p /mnt/huge1G
    mount -t hugetlbfs nodev /mnt/huge1G -o pagesize=1G
  fi
  echo $HUGE1G_NPAGES | tee /sys/devices/system/node/node*/hugepages/hugepages-1048576kB/nr_hugepages
fi

modprobe uio_pci_generic
modprobe igb_uio || true
