FROM ubuntu:18.04
RUN ( echo 'APT::Install-Recommends "no";'; echo 'APT::Install-Suggests "no";' ) >/etc/apt/apt.conf.d/80recommends && \
    apt-get update && \
    apt-get install -y -qq build-essential clang-8 curl git go-bindata libc6-dev-i386 libelf-dev libnuma-dev libssl-dev liburcu-dev pkg-config python3-distutils rake sudo && \
    curl -sL https://github.com/powerman/rpc-codec/releases/download/v1.1.3/jsonrpc2client-linux-x86_64 | install /dev/stdin /usr/local/bin/jsonrpc2client && \
    curl -sL https://deb.nodesource.com/setup_14.x | bash - && \
    apt-get install -y -qq nodejs clang-format-8 doxygen yamllint && \
    curl -sL https://bootstrap.pypa.io/get-pip.py | python3 && \
    pip install meson ninja && \
    curl -sL https://dl.google.com/go/go1.14.4.linux-amd64.tar.gz | tar -C /usr/local -xz && \
    curl -sL https://github.com/spdk/spdk/archive/v20.04.1.tar.gz | tar -C /root -xz && \
    cd /root/spdk-* && \
    ./scripts/pkgdep.sh && \
    apt-get clean
RUN curl -sL https://github.com/iovisor/ubpf/archive/089f6279752adfb01386600d119913403ed326ee.tar.gz | tar -C /root -xz && \
    cd /root/ubpf-*/vm && \
    make && \
    make install
RUN curl -sL http://archive.ubuntu.com/ubuntu/pool/universe/n/nasm/nasm_2.14.02.orig.tar.xz | tar -C /root -xJ && \
    cd /root/nasm-* && \
    ./configure && \
    make -j12 && \
    make install && \
    curl -sL https://github.com/intel/intel-ipsec-mb/archive/v0.53.tar.gz | tar -C /root -xz && \
    cd /root/intel-ipsec-mb-* && \
    make -j12 && \
    make install
RUN curl -sL https://static.dpdk.org/rel/dpdk-20.05.tar.xz | tar -C /root -xJ && \
    cd /root/dpdk-* && \
    meson -Dtests=false --libdir=lib build && \
    cd build && \
    ninja -j12 && \
    ninja install && \
    find /usr/local/lib -name 'librte_*.a' -delete && \
    ldconfig
RUN cd /root/spdk-* && \
    ./configure --enable-debug --disable-tests --with-shared --with-dpdk=/usr/local --without-vhost --without-isal --without-fuse && \
    make -j12 && \
    make install && \
    find /usr/local/lib -name 'libspdk_*.a' -delete && \
    ldconfig
ADD . /root/ndn-dpdk/
RUN export PATH=$PATH:/usr/local/go/bin && \
    cd /root/ndn-dpdk && \
    npm install && \
    make && \
    make install
