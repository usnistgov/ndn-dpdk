FROM debian:bookworm AS build
ARG APT_PKGS=
ARG DEPENDS_ENV=
ARG DEPENDS_ARGS=
ARG MAKE_ENV=
SHELL ["/bin/bash", "-c"]

RUN --mount=source=./docs/ndndpdk-depends.sh,target=/root/ndndpdk-depends.sh <<EOF
  set -euxo pipefail
  apt-get -y -qq update
  apt-get -y -qq install --no-install-recommends ca-certificates curl dpkg-dev gpg jq lsb-release ${APT_PKGS}
  env SKIPROOTCHECK=1 ${DEPENDS_ENV} /root/ndndpdk-depends.sh --dir=/root/ndndpdk-depends -y ${DEPENDS_ARGS}
  rm -rf /root/ndndpdk-depends
EOF

RUN --mount=rw,target=/root/ndn-dpdk/ <<EOF
  set -euxo pipefail
  export PATH="$PATH:/usr/local/go/bin"
  cd /root/ndn-dpdk
  corepack pnpm install
  env ${MAKE_ENV} make
  make install
EOF

RUN <<EOF
  set -euxo pipefail
  rm -rf \
    /usr/local/bin/__pycache__ \
    /usr/local/bin/meson \
    /usr/local/bin/readelf.py \
    /usr/local/bin/spdk_* \
    /usr/local/bin/wheel \
    /usr/local/go \
    /usr/local/include \
    /usr/local/lib/pkgconfig \
    /usr/local/lib/systemd \
    /usr/local/man \
    /usr/local/sbin \
    /usr/local/share/dpdk \
    /usr/local/share/man \
    /usr/local/share/xdp-tools \
    /usr/local/src
  mkdir -p /shlibdeps/debian
  cd /shlibdeps
  touch debian/control
  dpkg-shlibdeps --ignore-missing-info $(find /usr/local/lib -name '*.so') $(find /usr/local/bin -type f -executable) -O \
    | sed -n 's|^shlibs:Depends=||p' | sed 's| ([^)]*),\?||g' >/pkgs.txt
EOF


FROM debian:bookworm
SHELL ["/bin/bash", "-c"]

RUN --mount=from=build,source=/pkgs.txt,target=/pkgs.txt <<EOF
  apt-get -y -qq update
  apt-get -y -qq install --no-install-recommends iproute2 jq $(cat /pkgs.txt)
  rm -rf /var/lib/apt/lists/*
EOF

COPY --from=build /usr/local/ /usr/local/
RUN ldconfig

VOLUME /run/ndn
CMD ["/usr/local/bin/ndndpdk-svc", "--gqlserver", "http://:3030"]
