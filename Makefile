CLIBPREFIX=build/libndn-dpdk
INCLUDEFLAGS=-I/usr/local/include/dpdk -I/usr/include/dpdk
BPFPATH=build/strategy-bpf
BPFCC=clang-6.0
BPFFLAGS=-O2 -target bpf $(INCLUDEFLAGS) -Wno-int-to-void-pointer-cast

export CGO_CFLAGS_ALLOW='.*'
export CC_FOR_TARGET=${CC:-gcc}

all: gopkg tsc cmds

gopkg: godeps
	go build -v ./...

godeps: cbuilds cgoflags strategies app/version/version.go

cbuilds:
	bash -c "sed -n '/\.a:\s/ s/\$$(CLIBPREFIX)//p' Makefile | cut -d: -f1 | sed 's:^:$(CLIBPREFIX):' | xargs make"

cgoflags:
	bash -c "sed -n '/cgoflags\.go:/ p' Makefile | cut -d: -f1 | xargs make"

core/cgoflags.go:
	mk/make-cgoflags.sh core
	mk/make-cgoflags.sh core/coretest core

$(CLIBPREFIX)-core.a: core/*.h core/*.c
	mk/cbuild.sh core

core/running_stat/cgoflags.go:
	mk/make-cgoflags.sh core/running_stat

$(CLIBPREFIX)-urcu.a: core/urcu/*.h
	mk/cbuild.sh core/urcu

core/urcu/cgoflags.go:
	mk/make-cgoflags.sh core/urcu

$(CLIBPREFIX)-dpdk.a: $(CLIBPREFIX)-core.a dpdk/*.h dpdk/*.c
	mk/cbuild.sh dpdk

dpdk/cgostruct.go: dpdk/cgostruct.in.go
	cd dpdk; go tool cgo -godefs -- $(INCLUDEFLAGS) cgostruct.in.go | gofmt > cgostruct.go; rm -rf _obj

dpdk/cgoflags.go: dpdk/cgostruct.go
	mk/make-cgoflags.sh dpdk core
	mk/make-cgoflags.sh dpdk/dpdktest dpdk

$(CLIBPREFIX)-spdk.a: $(CLIBPREFIX)-dpdk.a spdk/*.h
	mk/cbuild.sh spdk

spdk/cgoflags.go: dpdk/cgoflags.go
	mk/make-cgoflags.sh spdk dpdk

ndn/error.go ndn/error.h: ndn/make-error.sh ndn/error.tsv
	ndn/make-error.sh

ndn/tlv-type.go ndn/tlv-type.h ndn/tlv-type.ts: ndn/make-tlv-type.sh ndn/tlv-type.tsv
	ndn/make-tlv-type.sh

$(CLIBPREFIX)-ndn.a: $(CLIBPREFIX)-dpdk.a ndn/*.h ndn/*.c ndn/error.h ndn/tlv-type.h
	mk/cbuild.sh ndn

ndn/cgoflags.go: dpdk/cgoflags.go
	mk/make-cgoflags.sh ndn dpdk

$(CLIBPREFIX)-iface.a: $(CLIBPREFIX)-ndn.a iface/*.h iface/*.c
	mk/cbuild.sh iface

iface/cgoflags.go: ndn/cgoflags.go
	mk/make-cgoflags.sh iface ndn
	mk/make-cgoflags.sh iface/ifacetest iface

iface/ethface/cgoflags.go: iface/cgoflags.go
	mk/make-cgoflags.sh iface/ethface iface

iface/socketface/cgoflags.go: iface/cgoflags.go
	mk/make-cgoflags.sh iface/socketface iface

iface/mockface/cgoflags.go: iface/cgoflags.go
	mk/make-cgoflags.sh iface/mockface iface

$(CLIBPREFIX)-mintmr.a: $(CLIBPREFIX)-dpdk.a container/mintmr/*.h container/mintmr/*.c
	mk/cbuild.sh container/mintmr

container/mintmr/cgoflags.go: dpdk/cgoflags.go
	mk/make-cgoflags.sh container/mintmr dpdk
	mk/make-cgoflags.sh container/mintmr/mintmrtest container/mintmr

$(CLIBPREFIX)-ndt.a: $(CLIBPREFIX)-ndn.a container/ndt/*.h container/ndt/*.c
	mk/cbuild.sh container/ndt

container/ndt/cgoflags.go: ndn/cgoflags.go
	mk/make-cgoflags.sh container/ndt ndn

$(CLIBPREFIX)-strategycode.a: container/strategycode/*.h container/strategycode/*.c
	mk/cbuild.sh container/strategycode

container/strategycode/cgoflags.go: core/cgoflags.go
	mk/make-cgoflags.sh container/strategycode core

$(CLIBPREFIX)-tsht.a: $(CLIBPREFIX)-dpdk.a $(CLIBPREFIX)-urcu.a container/tsht/*.h container/tsht/*.c
	mk/cbuild.sh container/tsht

container/tsht/cgoflags.go: dpdk/cgoflags.go core/urcu/cgoflags.go
	mk/make-cgoflags.sh container/tsht dpdk core/urcu

$(CLIBPREFIX)-fib.a: $(CLIBPREFIX)-tsht.a $(CLIBPREFIX)-ndt.a container/fib/*.h container/fib/*.c
	mk/cbuild.sh container/fib

container/fib/cgoflags.go: container/tsht/cgoflags.go container/strategycode/cgoflags.go ndn/cgoflags.go
	mk/make-cgoflags.sh container/fib container/strategycode container/tsht ndn

$(CLIBPREFIX)-diskstore.a: $(CLIBPREFIX)-ndn.a container/diskstore/*.h container/diskstore/*.c
	mk/cbuild.sh container/diskstore

container/diskstore/cgoflags.go: spdk/cgoflags.go ndn/cgoflags.go
	mk/make-cgoflags.sh container/diskstore spdk ndn

$(CLIBPREFIX)-pcct.a: $(CLIBPREFIX)-mintmr.a $(CLIBPREFIX)-fib.a container/pcct/*.h container/pcct/*.c
	mk/cbuild.sh container/pcct

container/pcct/cgoflags.go: container/mintmr/cgoflags.go container/fib/cgoflags.go
	mk/make-cgoflags.sh container/pcct container/mintmr container/fib
	mk/make-cgoflags.sh container/pit container/pcct
	mk/make-cgoflags.sh container/cs container/pcct

strategies: strategy/strategy_elf/bindata.go

strategy/strategy_elf/bindata.go: $(BPFPATH)/delay.o $(BPFPATH)/fastroute.o $(BPFPATH)/multicast.o $(BPFPATH)/reject.o $(BPFPATH)/roundrobin.o
	go-bindata -nomemcopy -pkg strategy_elf -prefix $(BPFPATH) -o /dev/stdout $(BPFPATH) | gofmt > strategy/strategy_elf/bindata.go

$(BPFPATH)/%.o: strategy/%.c $(CLIBPREFIX)-strategy.a
	mkdir -p $(BPFPATH)
	$(BPFCC) $(BPFFLAGS) -c $< -o $(BPFPATH)/$*.o

strategy-%.s: strategy/%.c
	$(BPFCC) $(BPFFLAGS) -c $< -S -o -

$(CLIBPREFIX)-strategy.a: strategy/api* $(CLIBPREFIX)-pcct.a
	mk/cbuild.sh strategy

appinit/cgoflags.go: dpdk/cgoflags.go
	mk/make-cgoflags.sh appinit dpdk

app/ndnping/cgoflags.go: iface/cgoflags.go
	mk/make-cgoflags.sh app/ndnping iface

$(CLIBPREFIX)-fwdp.a: $(CLIBPREFIX)-pcct.a $(CLIBPREFIX)-iface.a app/fwdp/*.h app/fwdp/*.c
	mk/cbuild.sh app/fwdp

app/fwdp/cgoflags.go: container/ndt/cgoflags.go container/fib/cgoflags.go container/pcct/cgoflags.go iface/cgoflags.go
	mk/make-cgoflags.sh app/fwdp container/ndt container/fib container/pcct iface

.PHONY: app/version/version.go
app/version/version.go:
	app/version/make-version.sh

.PHONY: tsdeps
tsdeps: ndn/tlv-type.ts mgmt/jrgen-spec-schema.ts

.PHONY: tsc
tsc: tsdeps
	node_modules/.bin/tsc

cmds: cmd-ndnfw-dpdk cmd-ndnping-dpdk

cmd-%: cmd/%/* godeps
	go install ./cmd/$*

install:
	mk/install.sh

uninstall:
	mk/uninstall.sh

doxygen:
	cd docs && doxygen Doxyfile 2>&1 | ./filter-Doxygen-warning.awk 1>&2

mgmtspec: docs/mgmtspec.json

docs/mgmtspec.json: tsc
	nodejs build/mgmt/make-spec >$@

.PHONY: docs
docs: doxygen mgmtspec

godoc:
	godoc -http ':6060' 2>/dev/null &

lint:
	mk/format-code.sh

test: godeps
	mk/gotest.sh

clean:
	awk '!(/node_modules/ || /\*\*/)' .dockerignore | xargs rm -rf
	awk 'BEGIN{FS="/"} $$1=="**"{print $$2}' .dockerignore | xargs -I{} -n1 find -name {} -delete
	go clean ./...
