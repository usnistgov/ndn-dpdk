FROM golang:1.20-bullseye AS build
ADD https://github.com/TomWright/dasel/releases/download/v2.1.1/dasel_linux_amd64 /dasel
RUN git clone https://github.com/named-data/YaNFD.git /app \
 && cd /app \
 && env CGO_ENABLED=0 make

FROM debian:bullseye
COPY --from=build /app/yanfd /dasel /usr/local/bin/
COPY --from=build /app/yanfd.toml.sample /usr/local/etc/ndn/
COPY start.sh /
RUN chmod +x /usr/local/bin/dasel /start.sh
CMD ["/start.sh"]
