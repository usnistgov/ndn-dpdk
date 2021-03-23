FROM ubuntu:focal
ARG APT_PKGS=
ARG DEPENDS_ARGS=
ARG MAKE_ENV=
ADD ./docs/ndndpdk-depends.sh /root/ndndpdk-depends.sh
RUN apt-get -y -qq update && \
    apt-get -y -qq install --no-install-recommends ca-certificates curl ${APT_PKGS} && \
    /root/ndndpdk-depends.sh --skiprootcheck --dir=/root/ndndpdk-depends -y ${DEPENDS_ARGS} && \
    apt-get -y -qq clean && \
    rm -rf /var/lib/apt/lists/* && \
    rm -rf /root/ndndpdk-depends
ADD . /root/ndn-dpdk/
RUN export PATH=$PATH:/usr/local/go/bin && \
    cd /root/ndn-dpdk && \
    npm install && \
    env ${MAKE_ENV} make && \
    make install
VOLUME /dev/hugepages /run/ndn
CMD ["/usr/local/sbin/ndndpdk-svc", "--gqlserver", "http://:3030"]
