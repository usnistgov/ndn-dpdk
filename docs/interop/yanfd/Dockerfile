FROM golang:1.16.4-buster AS build
RUN apt-get -y -qq update && \
    apt-get -y -qq install --no-install-recommends ca-certificates libpcap0.8-dev wget && \
    rm -rf /var/lib/apt/lists/*
RUN wget -q -O /dasel https://github.com/TomWright/dasel/releases/download/v1.15.0/dasel_linux_amd64 && \
    chmod +x /dasel
RUN git clone https://github.com/named-data/YaNFD.git /go/src/app && \
    cd /go/src/app && \
    env CGO_ENABLED=0 make

FROM debian:buster
RUN apt-get -y -qq update && \
    apt-get -y -qq install --no-install-recommends jq && \
    rm -rf /var/lib/apt/lists/*
COPY --from=build /go/src/app/yanfd /dasel /usr/local/bin/
COPY --from=build /go/src/app/yanfd.toml.sample /usr/local/etc/ndn/
COPY start.sh /
RUN chmod +x /start.sh
ENTRYPOINT ["/start.sh"]
