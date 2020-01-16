FROM ubuntu:18.04
RUN apt-get update && \
    apt-get install -y -qq build-essential clang-6.0 clang-format-6.0 curl doxygen gcc-7 git go-bindata libc6-dev-i386 libelf-dev libnuma-dev libssl-dev liburcu-dev rake socat sudo yamllint && \
    curl -L https://deb.nodesource.com/setup_12.x | bash - && \
    apt-get install -y -qq nodejs && \
    /usr/bin/npm install -g jayson && \
    curl -L https://dl.google.com/go/go1.13.5.linux-amd64.tar.gz | tar -C /usr/local -xz && \
    curl -L https://github.com/spdk/spdk/archive/v19.10.tar.gz | tar -C /tmp -xz && \
    cd /tmp/spdk-* && \
    sed '/libfuse3-dev/ d' ./scripts/pkgdep.sh | bash && \
    apt-get clean
RUN curl -L https://github.com/iovisor/ubpf/archive/644ad3ded2f015878f502765081e166ce8112baf.tar.gz | tar -C /tmp -xz && \
    cd /tmp/ubpf-*/vm && \
    make CC=gcc-7 && \
    mkdir -p /usr/local/include /usr/local/lib && \
    cp inc/ubpf.h /usr/local/include/ && \
    cp libubpf.a /usr/local/lib/
RUN curl -L http://archive.ubuntu.com/ubuntu/pool/universe/n/nasm/nasm_2.14.02.orig.tar.xz | tar -C /tmp -xJ && \
    cd /tmp/nasm-* && \
    ./configure && \
    make -j12 && \
    make install && \
    curl -L https://github.com/intel/intel-ipsec-mb/archive/v0.53.tar.gz | tar -C /tmp -xz && \
    cd /tmp/intel-ipsec-mb-* && \
    make -j12 && \
    make install
ADD . /root/go/src/ndn-dpdk/
RUN tar -C / -xzf /root/go/src/ndn-dpdk/kernel-headers.tgz && \
    curl -L http://fast.dpdk.org/rel/dpdk-19.11.tar.xz | tar -C /tmp -xJ && \
    cd /tmp/dpdk-* && \
    curl -L https://patches.dpdk.org/patch/63727/raw/ | patch -p1 && \
    make config T=x86_64-native-linuxapp-gcc && \
    sed -ri 's,(CONFIG_RTE_BUILD_SHARED_LIB=).*,\1y,' build/.config && \
    sed -ri 's,(CONFIG_RTE_LIBRTE_BPF_ELF=).*,\1y,' build/.config && \
    sed -ri 's,(CONFIG_RTE_LIBRTE_PMD_OPENSSL=).*,\1y,' build/.config && \
    sed -ri 's,(CONFIG_RTE_LIBRTE_PMD_AESNI_MB=).*,\1y,' build/.config && \
    make -j12 CC=gcc-7 EXTRA_CFLAGS=-g && \
    make install && \
    ldconfig
RUN cd /tmp/spdk-* && \
    CC=gcc-7 ./configure --enable-debug --with-shared --with-dpdk=/usr/local --without-vhost --without-isal --without-fuse && \
    make -j12 && \
    make install && \
    ldconfig
RUN export PATH=$PATH:/usr/local/go/bin && \
    export GOPATH=/root/go && \
    cd /root/go/src/ndn-dpdk && \
    npm install && \
    make godeps && \
    make goget && \
    make && \
    make install
