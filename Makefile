CLIBPREFIX=build-c/libndn-dpdk

all: cmds

cbuilds: $(CLIBPREFIX)-core.a $(CLIBPREFIX)-dpdk.a $(CLIBPREFIX)-ndn.a $(CLIBPREFIX)-nameset.a $(CLIBPREFIX)-pcct.a $(CLIBPREFIX)-iface.a

cmds: cmd-ndnpktcopy-dpdk cmd-ndnping-dpdk

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

ndn/namehash.h: ndn/namehash.c
	gcc -o /tmp/namehash.exe ndn/namehash.c -m64 -march=native -I/usr/local/include/dpdk -DNAMEHASH_GENERATOR
	openssl rand 16 | /tmp/namehash.exe > ndn/namehash.h
	rm /tmp/namehash.exe

ndn/tlv-type.go ndn/tlv-type.h: ndn/make-tlv-type.sh ndn/tlv-type.tsv
	ndn/make-tlv-type.sh

$(CLIBPREFIX)-ndn.a: $(CLIBPREFIX)-dpdk.a ndn/* ndn/error.h ndn/namehash.h ndn/tlv-type.h
	./build-c.sh ndn

go-ndn: $(CLIBPREFIX)-ndn.a ndn/error.go ndn/tlv-type.go
	go build ./ndn

$(CLIBPREFIX)-nameset.a: $(CLIBPREFIX)-ndn.a container/nameset/*
	./build-c.sh container/nameset

go-nameset: $(CLIBPREFIX)-nameset.a
	go build ./container/nameset

$(CLIBPREFIX)-ndt.a: $(CLIBPREFIX)-ndn.a container/ndt/*
	./build-c.sh container/ndt

go-ndt: $(CLIBPREFIX)-ndt.a
	go build ./container/ndt

$(CLIBPREFIX)-tsht.a: $(CLIBPREFIX)-dpdk.a container/tsht/*
	./build-c.sh container/tsht

$(CLIBPREFIX)-fib.a: $(CLIBPREFIX)-tsht.a $(CLIBPREFIX)-ndn.a container/fib/*
	./build-c.sh container/fib

go-fib: $(CLIBPREFIX)-fib.a
	go build ./container/fib

container/pcct/uthash.h: container/pcct/fetch-uthash.sh
	cd container/pcct && ./fetch-uthash.sh

$(CLIBPREFIX)-pcct.a: $(CLIBPREFIX)-ndn.a container/pcct/* container/pcct/uthash.h
	./build-c.sh container/pcct

go-pit: $(CLIBPREFIX)-pcct.a container/pit/*
	go build ./container/pit

go-cs: $(CLIBPREFIX)-pcct.a container/cs/*
	go build ./container/cs

$(CLIBPREFIX)-iface.a: $(CLIBPREFIX)-ndn.a iface/*
	./build-c.sh iface

go-iface: $(CLIBPREFIX)-iface.a iface/*
	go build ./iface

go-ethface: $(CLIBPREFIX)-iface.a iface/ethface/*
	go build ./iface/ethface

go-socketface: $(CLIBPREFIX)-iface.a iface/socketface/*
	go build ./iface/socketface

go-faceuri: $(CLIBPREFIX)-iface.a iface/faceuri/*
	go build ./iface/faceuri

go-appinit: appinit/*
	go build ./appinit

test:
	./gotest.sh
	integ/run.sh

clean:
	rm -rf build-c ndn/error.go ndn/error.h ndn/namehash.h ndn/tlv-type.go ndn/tlv-type.h
	go clean ./...

doxygen:
	cd docs && doxygen Doxyfile 2>&1 | ./filter-Doxygen-warning.awk 1>&2

dochttp: doxygen
	cd docs/html && python3 -m http.server 2>/dev/null &
