CLIBPREFIX=build/libndn-dpdk
STRATEGYPREFIX=build/strategy-bpf
BPFFLAGS=-O2 -target bpf -I/usr/local/include/dpdk -Wno-int-to-void-pointer-cast

all: cbuilds cgoflags strategies
	go build -v ./...

cbuilds:
	bash -c "sed -n '/\.a:\s/ s/\$$(CLIBPREFIX)//p' Makefile | cut -d: -f1 | sed 's:^:$(CLIBPREFIX):' | xargs make"

cgoflags:
	bash -c "sed -n '/cgoflags\.go:/ p' Makefile | cut -d: -f1 | xargs make"

core/cgoflags.go:
	./make-cgoflags.sh core

$(CLIBPREFIX)-core.a: core/*.h core/*.c
	./cbuild.sh core

core/running_stat/cgoflags.go:
	./make-cgoflags.sh core/running_stat

$(CLIBPREFIX)-urcu.a: core/urcu/*.h
	./cbuild.sh core/urcu

core/urcu/cgoflags.go:
	./make-cgoflags.sh core/urcu

$(CLIBPREFIX)-dpdk.a: $(CLIBPREFIX)-core.a dpdk/*.h dpdk/*.c
	./cbuild.sh dpdk

dpdk/cgostruct.go: dpdk/cgostruct.in.go
	cd dpdk; go tool cgo -godefs -- -I/usr/local/include/dpdk cgostruct.in.go | gofmt > cgostruct.go; rm -rf _obj

dpdk/cgoflags.go: dpdk/cgostruct.go
	./make-cgoflags.sh dpdk core
	./make-cgoflags.sh dpdk/dpdktest dpdk

ndn/error.go ndn/error.h: ndn/make-error.sh ndn/error.tsv
	ndn/make-error.sh

ndn/namehash.h: ndn/namehash.c
	gcc -o /tmp/namehash.exe ndn/namehash.c -m64 -march=native -I/usr/local/include/dpdk -DNAMEHASH_GENERATOR
	openssl rand 16 | /tmp/namehash.exe > ndn/namehash.h
	rm /tmp/namehash.exe

ndn/tlv-type.go ndn/tlv-type.h: ndn/make-tlv-type.sh ndn/tlv-type.tsv
	ndn/make-tlv-type.sh

$(CLIBPREFIX)-ndn.a: $(CLIBPREFIX)-dpdk.a ndn/*.h ndn/*.c ndn/error.h ndn/namehash.h ndn/tlv-type.h
	./cbuild.sh ndn

ndn/cgoflags.go: dpdk/cgoflags.go
	./make-cgoflags.sh ndn dpdk

$(CLIBPREFIX)-iface.a: $(CLIBPREFIX)-ndn.a iface/*.h iface/*.c
	./cbuild.sh iface

iface/cgoflags.go: ndn/cgoflags.go
	./make-cgoflags.sh iface ndn
	./make-cgoflags.sh iface/ifacetest iface

iface/ethface/cgoflags.go: iface/cgoflags.go
	./make-cgoflags.sh iface/ethface iface

iface/socketface/cgoflags.go: iface/cgoflags.go
	./make-cgoflags.sh iface/socketface iface

iface/mockface/cgoflags.go: iface/cgoflags.go
	./make-cgoflags.sh iface/mockface iface

$(CLIBPREFIX)-mintmr.a: $(CLIBPREFIX)-dpdk.a container/mintmr/*.h container/mintmr/*.c
	./cbuild.sh container/mintmr

container/mintmr/cgoflags.go: dpdk/cgoflags.go
	./make-cgoflags.sh container/mintmr dpdk

$(CLIBPREFIX)-nameset.a: $(CLIBPREFIX)-ndn.a container/nameset/*.h container/nameset/*.c
	./cbuild.sh container/nameset

container/nameset/cgoflags.go: ndn/cgoflags.go
	./make-cgoflags.sh container/nameset ndn

$(CLIBPREFIX)-ndt.a: $(CLIBPREFIX)-ndn.a container/ndt/*.h container/ndt/*.c
	./cbuild.sh container/ndt

container/ndt/cgoflags.go: ndn/cgoflags.go
	./make-cgoflags.sh container/ndt ndn

$(CLIBPREFIX)-tsht.a: $(CLIBPREFIX)-dpdk.a $(CLIBPREFIX)-urcu.a container/tsht/*.h container/tsht/*.c
	./cbuild.sh container/tsht

container/tsht/cgoflags.go: dpdk/cgoflags.go core/urcu/cgoflags.go
	./make-cgoflags.sh container/tsht dpdk core/urcu

$(CLIBPREFIX)-fib.a: $(CLIBPREFIX)-tsht.a $(CLIBPREFIX)-ndn.a container/fib/*.h container/fib/*.c
	./cbuild.sh container/fib

container/fib/cgoflags.go: container/tsht/cgoflags.go ndn/cgoflags.go
	./make-cgoflags.sh container/fib container/tsht ndn

$(CLIBPREFIX)-pcct.a: $(CLIBPREFIX)-mintmr.a $(CLIBPREFIX)-ndn.a container/pcct/*.h container/pcct/*.c
	./cbuild.sh container/pcct

container/pcct/cgoflags.go: container/mintmr/cgoflags.go ndn/cgoflags.go
	./make-cgoflags.sh container/pcct container/mintmr ndn
	./make-cgoflags.sh container/pit container/pcct
	./make-cgoflags.sh container/cs container/pcct

strategies: strategy/strategy_elf/bindata.go

strategy/strategy_elf/bindata.go: $(STRATEGYPREFIX)/multicast.o $(STRATEGYPREFIX)/roundrobin.o
	go-bindata -nomemcopy -pkg strategy_elf -prefix $(STRATEGYPREFIX) -o /dev/stdout $(STRATEGYPREFIX) | gofmt > strategy/strategy_elf/bindata.go

strategy-% $(STRATEGYPREFIX)/%.o: strategy/%.c $(CLIBPREFIX)-strategy.a
	mkdir -p $(STRATEGYPREFIX)
	clang $(BPFFLAGS) -c $< -o $(STRATEGYPREFIX)/$*.o

$(CLIBPREFIX)-strategy.a: strategy/api*
	./cbuild.sh strategy

appinit/cgoflags.go: dpdk/cgoflags.go
	./make-cgoflags.sh appinit dpdk

app/ndnping/cgoflags.go: container/nameset/cgoflags.go iface/cgoflags.go
	./make-cgoflags.sh app/ndnping container/nameset iface

$(CLIBPREFIX)-fwdp.a: $(CLIBPREFIX)-ndt.a $(CLIBPREFIX)-fib.a $(CLIBPREFIX)-pcct.a $(CLIBPREFIX)-iface.a app/fwdp/*.h app/fwdp/*.c
	./cbuild.sh app/fwdp

app/fwdp/cgoflags.go: container/ndt/cgoflags.go container/fib/cgoflags.go container/pcct/cgoflags.go iface/cgoflags.go
	./make-cgoflags.sh app/fwdp container/ndt container/fib container/pcct iface

.PHONY: app/version/version.go
app/version/version.go:
	app/version/make-version.sh

cmd-deps: cbuilds cgoflags app/version/version.go

cmds: cmd-ndnfw-dpdk cmd-ndnping-dpdk cmd-ndnpktcopy-dpdk

cmd-%: cmd/%/* cmd-deps
	go install ./cmd/$*

cmd-ndnfw-dpdk: cmd/ndnfw-dpdk/* cmd-deps strategies
	go install ./cmd/ndnfw-dpdk

mgmtclient: cmd/mgmtclient/*
	mkdir -p build
	cd build && rm -f mgmt*.sh
	cd cmd/mgmtclient && cp mgmt*.sh ../../build/
	chmod +x build/mgmt*.sh

cmd/ndnpktcopy-dpdk/cgoflags.go: iface/cgoflags.go
	./make-cgoflags.sh cmd/ndnpktcopy-dpdk iface

integ/core/siphash/cgoflags.go:
	./make-cgoflags.sh integ/core/siphash core

integ/container/mintmr/cgoflags.go:
	./make-cgoflags.sh integ/container/mintmr container/mintmr

test: cbuilds cgoflags
	./gotest.sh
	integ/run.sh

doxygen:
	cd docs && doxygen Doxyfile 2>&1 | ./filter-Doxygen-warning.awk 1>&2

codedoc:
	bash docs/codedoc.sh

docs/mgmtschema/schema.json: docs/mgmtschema/*.js
	nodejs docs/mgmtschema/ > docs/mgmtschema/schema.json

dochttp: doxygen codedoc docs/mgmtschema/schema.json
	cd docs && python3 -m http.server 2>/dev/null &

godochttp:
	godoc -http ':6060' 2>/dev/null &

clean:
	rm -rf build docs/doxygen docs/codedoc ndn/error.go ndn/error.h ndn/namehash.h ndn/tlv-type.go ndn/tlv-type.h strategy/strategy_elf/bindata.go
	find -name 'cgoflags.go' -o -name 'cgostruct.go' -delete
	go clean ./...
