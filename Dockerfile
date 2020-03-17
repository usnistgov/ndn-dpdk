FROM ubuntu:18.04
RUN apt-get update && \
    apt-get install -y -qq build-essential clang-6.0 curl gcc-7 git go-bindata libc6-dev-i386 libelf-dev libnuma-dev libssl-dev liburcu-dev pkg-config python3.8 python3-distutils rake sudo && \
    curl -L https://github.com/powerman/rpc-codec/releases/download/v1.1.3/jsonrpc2client-linux-x86_64 | install /dev/stdin /usr/local/bin/jsonrpc2client && \
    curl -L https://deb.nodesource.com/setup_12.x | bash - && \
    apt-get install -y -qq nodejs clang-format-6.0 doxygen yamllint && \
    curl https://bootstrap.pypa.io/get-pip.py -o get-pip.py && \
    python3.8 get-pip.py && \
    pip install meson ninja && \
    curl -L https://dl.google.com/go/go1.14.linux-amd64.tar.gz | tar -C /usr/local -xz && \
    curl -L https://github.com/spdk/spdk/archive/v19.10.1.tar.gz | tar -C /tmp -xz && \
    cd /tmp/spdk-* && \
    sed '/libfuse3-dev/ d' ./scripts/pkgdep.sh | bash && \
    apt-get clean
RUN curl -L https://github.com/iovisor/ubpf/archive/9d10569738730a7357c706bf1d62c53ef5d329ad.tar.gz | tar -C /tmp -xz && \
    cd /tmp/ubpf-*/vm && \
    make CC=gcc-7 && \
    make install
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
    curl -L https://static.dpdk.org/rel/dpdk-19.11.tar.xz | tar -C /tmp -xJ && \
    cd /tmp/dpdk-* && \
    curl -L https://patches.dpdk.org/patch/65156/raw/ | patch -p1 && \
    curl -L https://patches.dpdk.org/patch/65158/raw/ | patch -p1 && \
    curl -L https://patches.dpdk.org/patch/65270/raw/ | patch -p1 && \
    CC=gcc-7 meson -Dtests=false --libdir=lib build && \
    cd build && \
    ninja && \
    ninja install && \
    find /usr/local/lib -name 'librte_*.a' -delete && \
    ldconfig
RUN cd /tmp/spdk-* && \
    CC=gcc-7 ./configure --enable-debug --disable-tests --with-shared --with-dpdk=/usr/local --without-vhost --without-isal --without-fuse && \
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
