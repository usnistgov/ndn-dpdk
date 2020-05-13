FROM ubuntu:18.04
RUN ( echo 'APT::Install-Recommends "no";'; echo 'APT::Install-Suggests "no";' ) >/etc/apt/apt.conf.d/80recommends && \
    apt-get update && \
    apt-get install -y -qq build-essential clang-6.0 curl git go-bindata libc6-dev-i386 libelf-dev libnuma-dev libssl-dev liburcu-dev pkg-config python3-distutils rake sudo && \
    curl -L https://github.com/powerman/rpc-codec/releases/download/v1.1.3/jsonrpc2client-linux-x86_64 | install /dev/stdin /usr/local/bin/jsonrpc2client && \
    curl -L https://deb.nodesource.com/setup_12.x | bash - && \
    apt-get install -y -qq nodejs clang-format-6.0 doxygen yamllint && \
    curl https://bootstrap.pypa.io/get-pip.py | python3 && \
    pip install meson ninja && \
    curl -L https://dl.google.com/go/go1.14.2.linux-amd64.tar.gz | tar -C /usr/local -xz && \
    curl -L https://github.com/spdk/spdk/archive/v19.10.1.tar.gz | tar -C /root -xz && \
    cd /root/spdk-* && \
    sed '/libfuse3-dev/ d' ./scripts/pkgdep.sh | bash - && \
    apt-get clean
RUN curl -L https://github.com/iovisor/ubpf/archive/4cbf7998e6f72f3f4d0b30cf30cb508428eb421f.tar.gz | tar -C /root -xz && \
    cd /root/ubpf-*/vm && \
    make && \
    make install
RUN curl -L http://archive.ubuntu.com/ubuntu/pool/universe/n/nasm/nasm_2.14.02.orig.tar.xz | tar -C /root -xJ && \
    cd /root/nasm-* && \
    ./configure && \
    make -j12 && \
    make install && \
    curl -L https://github.com/intel/intel-ipsec-mb/archive/v0.53.tar.gz | tar -C /root -xz && \
    cd /root/intel-ipsec-mb-* && \
    make -j12 && \
    make install
RUN curl -L https://static.dpdk.org/rel/dpdk-19.11.tar.xz | tar -C /root -xJ && \
    cd /root/dpdk-* && \
    curl -L https://patches.dpdk.org/patch/65156/raw/ | patch -p1 && \
    curl -L https://patches.dpdk.org/patch/65158/raw/ | patch -p1 && \
    curl -L https://patches.dpdk.org/patch/65270/raw/ | patch -p1 && \
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
