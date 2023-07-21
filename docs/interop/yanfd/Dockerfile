FROM golang:1.20-bookworm AS build
RUN git clone https://github.com/named-data/YaNFD.git /app \
 && cd /app \
 && env CGO_ENABLED=0 make

FROM debian:bookworm
ADD https://github.com/TomWright/dasel/releases/download/v2.3.4/dasel_linux_amd64 /usr/local/bin/dasel
COPY --from=build /app/yanfd /usr/local/bin/
COPY --from=build /app/yanfd.toml.sample /usr/local/etc/ndn/
COPY start.sh /
RUN chmod +x /usr/local/bin/dasel /start.sh
CMD ["/start.sh"]
