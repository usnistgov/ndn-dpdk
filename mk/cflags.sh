export CC=${CC:-gcc}
export CGO_CFLAGS_ALLOW='.*'
CGO_CFLAGS='-Werror -Wno-error=deprecated-declarations -m64 -pthread -O3 -g '$(pkg-config --cflags libdpdk | sed 's/-include [^ ]*//')
CGO_LIBS='-L/usr/local/lib -lurcu-qsbr -lurcu-cds -lubpf -lspdk -lspdk_env_dpdk -lrte_bus_pci -lrte_bus_vdev -lrte_pmd_ring '$(pkg-config --libs libdpdk)' -lnuma -lm'

if [[ -n $RELEASE ]]; then
  export CFLAGS='-DNDEBUG -DZF_LOG_DEF_LEVEL=ZF_LOG_INFO'
  CGO_CFLAGS=$CGO_CFLAGS' '$CFLAGS
fi
