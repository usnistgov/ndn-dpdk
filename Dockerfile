FROM ubuntu:18.04
RUN ( echo 'APT::Install-Recommends "no";'; echo 'APT::Install-Suggests "no";' ) >/etc/apt/apt.conf.d/80custom && \
    apt-get update && \
    apt-get install -y -qq build-essential ca-certificates clang-8 curl git libc6-dev-i386 libelf-dev libnuma-dev libssl-dev liburcu-dev pkg-config python3-distutils sudo && \
    curl -sL https://deb.nodesource.com/setup_14.x | bash - && \
    apt-get install -y -qq clang-format-8 doxygen nodejs yamllint && \
    sudo update-alternatives --install /usr/bin/python python /usr/bin/python3 1 && \
    curl -sL https://bootstrap.pypa.io/get-pip.py | python && \
    pip install -U meson ninja && \
    curl -sL https://dl.google.com/go/go1.15.2.linux-amd64.tar.gz | tar -C /usr/local -xz && \
    curl -sL https://github.com/powerman/rpc-codec/releases/download/v1.1.3/jsonrpc2client-linux-x86_64 | install /dev/stdin /usr/local/bin/jsonrpc2client && \
    curl -sL https://github.com/spdk/spdk/archive/v20.07.tar.gz | tar -C /root -xz && \
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
    make -j$(nproc) && \
    make install && \
    curl -sL https://github.com/intel/intel-ipsec-mb/archive/v0.54.tar.gz | tar -C /root -xz && \
    cd /root/intel-ipsec-mb-* && \
    make -j$(nproc) && \
    make install
RUN curl -sL https://static.dpdk.org/rel/dpdk-20.08.tar.xz | tar -C /root -xJ && \
    cd /root/dpdk-* && \
    meson -Dtests=false --libdir=lib build && \
    cd build && \
    ninja -j$(nproc) && \
    ninja install && \
    find /usr/local/lib -name 'librte_*.a' -delete && \
    ldconfig
RUN cd /root/spdk-* && \
    ./configure --enable-debug --disable-tests --with-shared --with-dpdk=/usr/local --without-vhost --without-isal --without-fuse && \
    make -j$(nproc) && \
    make install && \
    find /usr/local/lib -name 'libspdk_*.a' -delete && \
    ldconfig
ADD . /root/ndn-dpdk/
RUN export PATH=$PATH:/usr/local/go/bin && \
    cd /root/ndn-dpdk && \
    npm install && \
    make && \
    make install
