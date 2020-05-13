export CC=${CC:-gcc}
export CGO_CFLAGS_ALLOW='.*'
CFLAGS='-Werror -Wno-error=deprecated-declarations -m64 -pthread -O3 -g '$(pkg-config --cflags libdpdk | sed 's/-include [^ ]*//')
LIBS='-L/usr/local/lib -lurcu-qsbr -lurcu-cds -lubpf -lspdk -lspdk_env_dpdk -lrte_bus_pci -lrte_bus_vdev -lrte_pmd_ring '$(pkg-config --libs libdpdk)' -lnuma -lm'

if ! [[ $MK_CGOFLAGS ]]; then
  CFLAGS='-Wall '$CFLAGS
fi

if [[ -n $RELEASE ]]; then
  CFLAGS=$CFLAGS' -DNDEBUG -DZF_LOG_DEF_LEVEL=ZF_LOG_INFO'
fi

if [[ $CC =~ clang ]]; then
  CFLAGS=$CFLAGS' -Wno-error=address-of-packed-member'
fi
