CFLAGS='-Werror -Wno-error=deprecated-declarations -m64 -pthread -O3 -g -march=native -I/usr/local/include/dpdk -I/usr/include/dpdk'
LIBS='-L/usr/local/lib -lurcu-qsbr -lurcu-cds -lubpf -lspdk -lspdk_env_dpdk -ldpdk -lnuma -lm'

if [[ -n $RELEASE ]]; then
  CFLAGS=$CFLAGS' -DNDEBUG -DZF_LOG_DEF_LEVEL=ZF_LOG_INFO'
fi

if [[ $CC == 'clang' ]]; then
  CFLAGS=$CFLAGS' -Wno-error=address-of-packed-member -Wno-error=initializer-overrides'
fi
