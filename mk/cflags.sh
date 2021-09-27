export CC=${CC:-gcc}
export CGO_CFLAGS_ALLOW='.*'

MESONFLAGS=''
CFLAGS='-Wno-unused-function -Wno-unused-parameter -Wno-missing-braces -D_GNU_SOURCE'
LGCOV=''
if [[ $NDNDPDK_MK_RELEASE -eq 1 ]]; then
  CFLAGS=$CFLAGS' -DNDEBUG -DN_LOG_LEVEL=RTE_LOG_NOTICE'
fi
if [[ $NDNDPDK_MK_THREADSLEEP -eq 1 ]]; then
  CFLAGS=$CFLAGS' -DNDNDPDK_THREADSLEEP'
fi
if [[ $NDNDPDK_MK_COVERAGE -eq 1 ]]; then
  MESONFLAGS=$MESONFLAGS' -Db_coverage=true'
  LGCOV='-lgcov'
fi

export CFLAGS
CGO_CFLAGS="-Werror $CFLAGS -m64 -pthread -O3 -g $(pkg-config --cflags libdpdk liburing | sed 's/-include [^ ]*//g')"
CGO_LIBS="-L/usr/local/lib $LGCOV -lurcu-qsbr -lurcu-cds -lubpf $(pkg-config --libs spdk_bdev spdk_init spdk_env_dpdk) -lrte_bus_pci -lrte_bus_vdev -lrte_net_ring $(pkg-config --libs libdpdk liburing) -lnuma -lm"
