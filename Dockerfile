FROM ubuntu:18.04
RUN apt-get update && \
    apt-get install -y -qq clang-3.9 clang-format-3.9 curl doxygen dpdk-dev git go-bindata libc6-dev-i386 libnuma-dev liburcu-dev pandoc socat sudo yamllint
RUN curl -L https://dl.google.com/go/go1.10.2.linux-amd64.tar.gz | sudo tar -C /usr/local -xz
RUN cd /tmp && \
    curl -L https://github.com/iovisor/ubpf/archive/10e0a45b11ea27696add38c33e24dbc631caffb6.tar.gz | tar xz && \
    cd ubpf-*/vm && \
    make && \
    sudo mkdir -p /usr/local/include /usr/local/lib && \
    sudo cp inc/ubpf.h /usr/local/include/ && \
    sudo cp libubpf.a /usr/local/lib/
RUN curl -sL https://deb.nodesource.com/setup_8.x | sudo -E bash - && \
    apt-get install -y -qq nodejs && \
    npm install -g jayson
ADD . /root/go/src/ndn-dpdk/
RUN export PATH=$PATH:/usr/local/go/bin && \
    export GOPATH=/root/go && \
    cd /root/go/src/ndn-dpdk && \
    make godeps && \
    go get -d -t ./... && \
    make all cmds
