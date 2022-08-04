FROM golang:1.19-bullseye AS build
RUN apt-get -y -qq update && \
    apt-get -y -qq install --no-install-recommends ca-certificates wget && \
    rm -rf /var/lib/apt/lists/*
RUN wget -q -O /dasel https://github.com/TomWright/dasel/releases/download/v1.24.3/dasel_linux_amd64 && \
    chmod +x /dasel
RUN git clone https://github.com/named-data/YaNFD.git /app && \
    cd /app && \
    env CGO_ENABLED=0 make

FROM debian:bullseye
COPY --from=build /app/yanfd /dasel /usr/local/bin/
COPY --from=build /app/yanfd.toml.sample /usr/local/etc/ndn/
COPY start.sh /
RUN chmod +x /start.sh
ENTRYPOINT ["/start.sh"]
