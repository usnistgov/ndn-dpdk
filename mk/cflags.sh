export CC=${CC:-gcc}
export CGO_CFLAGS_ALLOW='.*'

CFLAGS='-Wno-unused-function -Wno-unused-parameter -Wno-missing-braces -D_DEFAULT_SOURCE'
if [[ $NDNDPDK_MK_RELEASE -eq 1 ]]; then
  CFLAGS=$CFLAGS' -DNDEBUG -DN_LOG_LEVEL=RTE_LOG_NOTICE'
else
  CFLAGS=$CFLAGS' -DN_LOG_LEVEL=RTE_LOG_DEBUG'
fi
if [[ $NDNDPDK_MK_THREADSLEEP -eq 1 ]]; then
  CFLAGS=$CFLAGS' -DNDNDPDK_THREADSLEEP'
fi

export CFLAGS
CGO_CFLAGS="-Werror $CFLAGS -m64 -pthread -O3 -g $(pkg-config --cflags libdpdk liburing | sed 's/-include [^ ]*//')"
CGO_LIBS="-L/usr/local/lib -lurcu-qsbr -lurcu-cds -lubpf $(pkg-config --libs spdk_bdev spdk_init spdk_env_dpdk) -lrte_bus_pci -lrte_bus_vdev -lrte_net_ring $(pkg-config --libs libdpdk liburing) -lnuma -lm"
