export CC=${CC:-gcc}
export CGO_CFLAGS_ALLOW='.*'

CFLAGS='-Wno-unused-function -Wno-unused-parameter -Wno-missing-braces'
if [[ -n $RELEASE ]]; then
  CFLAGS=$CFLAGS' -DNDEBUG -DZF_LOG_DEF_LEVEL=ZF_LOG_INFO'
fi

export CFLAGS
CGO_CFLAGS='-Werror '$CFLAGS' -m64 -pthread -O3 -g '$(pkg-config --cflags libdpdk | sed 's/-include [^ ]*//')
CGO_LIBS='-L/usr/local/lib -lurcu-qsbr -lurcu-cds -lubpf -lspdk -lspdk_env_dpdk -lrte_bus_pci -lrte_bus_vdev -lrte_net_ring '$(pkg-config --libs libdpdk)' -lnuma -lm'
