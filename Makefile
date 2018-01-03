CLIBPREFIX=build-c/libndn-dpdk

all: cmds

gopkgs: go-dpdk go-ndn go-iface go-ethface go-socketface

cbuilds: $(CLIBPREFIX)-core.a $(CLIBPREFIX)-dpdk.a $(CLIBPREFIX)-ndn.a $(CLIBPREFIX)-iface.a

cmds: cmd-ndndump-dpdk cmd-ndnpktcopy-dpdk

cmd-%: cmd/%/* cbuilds
	go install ./cmd/$*

$(CLIBPREFIX)-core.a: core/*
	./build-c.sh core

$(CLIBPREFIX)-dpdk.a: $(CLIBPREFIX)-core.a dpdk/*
	./build-c.sh dpdk

go-dpdk: $(CLIBPREFIX)-dpdk.a
	go build ./dpdk

ndn/error.go ndn/error.h: ndn/make-error.sh ndn/error.tsv
	ndn/make-error.sh

ndn/tlv-type.go ndn/tlv-type.h: ndn/make-tlv-type.sh ndn/tlv-type.tsv
	ndn/make-tlv-type.sh

$(CLIBPREFIX)-ndn.a: $(CLIBPREFIX)-dpdk.a ndn/* ndn/error.h ndn/tlv-type.h
	./build-c.sh ndn

go-ndn: $(CLIBPREFIX)-ndn.a ndn/error.go ndn/tlv-type.go
	go build ./ndn

$(CLIBPREFIX)-iface.a: $(CLIBPREFIX)-ndn.a iface/*
	./build-c.sh iface

go-iface: $(CLIBPREFIX)-iface.a
	go build ./iface

go-ethface: $(CLIBPREFIX)-iface.a iface/ethface/*
	go build ./iface/ethface

go-socketface: $(CLIBPREFIX)-iface.a iface/socketface/*
	go build ./iface/socketface

unittest:
	./gotest.sh dpdk/dpdktest
	./gotest.sh ndn
	./gotest.sh iface/ethface

test: unittest
	integ/run.sh

clean:
	rm -rf build-c ndn/error.go ndn/error.h ndn/tlv-type.go ndn/tlv-type.h
	go clean ./...

doxygen:
	cd docs && doxygen Doxyfile 2>&1 | ./filter-Doxygen-warning.awk 1>&2

dochttp: doxygen
	cd docs/html && python3 -m http.server 2>/dev/null &
