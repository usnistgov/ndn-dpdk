export CGO_CFLAGS_ALLOW := '.*'
ifeq ($(origin CC),default)
	CC = gcc
endif
export CC

all: gopkg npm cmds

gopkg: godeps
	go build -v ./...

godeps: app/version/version.go ndni/enum_string.go build/libndn-dpdk-c.a build/cgoflags.done build/cgostruct.done
	rake strategies

.PHONY: app/version/version.go
app/version/version.go:
	app/version/make-version.sh

csrc/ndn/an.h: ndn/an/*.go
	go run ./mk/enumgen/ -type=TlvType,NackReason -guard=NDN_DPDK_NDN_AN_H -out=$@ ./$(<D)

csrc/ndn/enum.h ndni/enum_string.go: ndni/enum.go
	mk/gogenerate.sh ./$(@D)

.PHONY: build/libndn-dpdk-c.a
build/libndn-dpdk-c.a: build/build.ninja csrc/ndn/an.h csrc/ndn/enum.h
	cd build && ninja

build/cgoflags.done: build/build.ninja
	cd build && ninja cgoflags

build/cgostruct.done: build/build.ninja
	cd build && ninja cgostruct

build/build.ninja: meson.build csrc/meson.build mk/meson.build
	bash -c 'source mk/cflags.sh; meson build'

csrc/meson.build mk/meson.build:
	mk/update-list.sh

.PHONY: tsc
tsc:
	node_modules/.bin/tsc

.PHONY: npm
npm: tsc
	mv $$(npm pack -s .) build/ndn-dpdk.tgz

.PHONY: cmds
cmds: build/bin/ndnfw-dpdk build/bin/ndnping-dpdk build/bin/ndndpdk-hrlog2histogram

build/bin/%: cmd/%/* godeps
	GOBIN=$$(realpath build/bin) go install ./cmd/$*

install:
	mk/install.sh

uninstall:
	mk/uninstall.sh

doxygen:
	cd docs && doxygen Doxyfile 2>&1 | ./filter-Doxygen-warning.awk 1>&2

mgmtspec: docs/mgmtspec.json

docs/mgmtspec.json:
	./node_modules/.bin/ts-node mgmt/make-spec.ts >$@

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
