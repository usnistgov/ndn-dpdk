FROM debian:buster
ADD ./docs/ndndpdk-depends.sh /root/ndndpdk-depends.sh
RUN apt-get -y -qq update && \
    apt-get -y -qq install curl sudo && \
    /root/ndndpdk-depends.sh --skiprootcheck --dir=/root/ndndpdk-depends -y && \
    apt-get -y -qq clean && \
    rm -rf /root/ndndpdk-depends
ADD . /root/ndn-dpdk/
RUN export PATH=$PATH:/usr/local/go/bin && \
    cd /root/ndn-dpdk && \
    npm install && \
    make && \
    make install
