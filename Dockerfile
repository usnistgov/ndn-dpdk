FROM ubuntu:focal
ARG APT_PKGS=
ARG DEPENDS_ENV=
ARG DEPENDS_ARGS=
ARG MAKE_ENV=
COPY ./docs/ndndpdk-depends.sh /root/ndndpdk-depends.sh
RUN apt-get -y -qq update && \
    apt-get -y -qq install --no-install-recommends ca-certificates curl iproute2 jq ${APT_PKGS} && \
    env ${DEPENDS_ENV} /root/ndndpdk-depends.sh --skiprootcheck --dir=/root/ndndpdk-depends -y ${DEPENDS_ARGS} && \
    rm -rf /var/lib/apt/lists/* /root/ndndpdk-depends /root/ndndpdk-depends.sh
COPY . /root/ndn-dpdk/
RUN export PATH=$PATH:/usr/local/go/bin && \
    cd /root/ndn-dpdk && \
    npm install && \
    env ${MAKE_ENV} make && \
    make install
VOLUME /dev/hugepages /run/ndn
CMD ["/usr/local/sbin/ndndpdk-svc", "--gqlserver", "http://:3030"]
