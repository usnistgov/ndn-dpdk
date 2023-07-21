FROM debian:bookworm
RUN apt-get -y -qq update \
 && apt-get -y -qq install --no-install-recommends ca-certificates \
 && echo "deb [arch=amd64 trusted=yes] https://nfd-nightly-apt.ndn.today/debian bookworm main" > /etc/apt/sources.list.d/nfd-nightly.list \
 && apt-get -y -qq update \
 && apt-get -y -qq install --no-install-recommends infoedit ndnpeek ndnping ndnsec nfd \
 && rm -rf /var/lib/apt/lists/*
COPY start.sh /
RUN chmod +x /start.sh \
 && setcap -r /usr/bin/nfd || true \
 && echo 'transport=unix:///run/ndn/nfd.sock' > /etc/ndn/client.conf
CMD ["/start.sh"]
