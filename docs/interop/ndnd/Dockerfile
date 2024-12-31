FROM golang:1.23-alpine3.21 AS build
SHELL ["/bin/ash", "-c"]
ADD https://github.com/named-data/ndnd/archive/ba1278f199c58a2072f29e95a3a5587803e294f8.zip /ndnd.zip

RUN <<EOF
  set -euxo pipefail
  cd /
  unzip -n ndnd.zip
  mv ndnd-* ndnd
  cd /ndnd
  env CGO_ENABLED=0 GOBIN=/build go install ./cmd/ndnd
EOF

ADD --chmod=755 https://github.com/mikefarah/yq/releases/download/v4.44.6/yq_linux_amd64 /usr/local/bin/yq

RUN <<EOF
  set -euxo pipefail
  yq -oj -P '
    ... comments="" |
    .faces.websocket.enabled=false |
    .fw.threads=2 |
    .tables.rib.readvertise_nlsr=false |
    sort_keys(..)
  ' /ndnd/fw/yanfd.sample.yml > /build/yanfd.yml
EOF

FROM scratch
COPY --from=build /build/* /
ENTRYPOINT ["/ndnd"]
CMD ["fw", "run", "/yanfd.yml"]
