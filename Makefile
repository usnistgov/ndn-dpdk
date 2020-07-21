export CGO_CFLAGS_ALLOW := '.*'
ifeq ($(origin CC),default)
	CC = gcc
endif
export CC

all: gopkg npm cmds

gopkg: godeps
	go build -v ./...

godeps: app/version/version.go strategy/strategyelf/bindata.go build/libndn-dpdk-c.a build/cgodeps.done

.PHONY: app/version/version.go
app/version/version.go:
	app/version/make-version.sh

csrc/ndni/enum.h csrc/ndni/an.h: ndni/enum.go ndn/an/*.go
	mk/gogenerate.sh ./$(<D)

csrc/iface/enum.h: iface/enum.go
	mk/gogenerate.sh ./$(<D)

csrc/pcct/cs-enum.h: container/cs/enum.go
	mk/gogenerate.sh ./$(<D)

ndni/ndnitest/cgo_test.go: ndni/ndnitest/*_ctest.go
	mk/gogenerate.sh ./$(<D)

strategy/strategyelf/bindata.go: strategy/*.c
	mk/gogenerate.sh ./$(@D)

.PHONY: build/libndn-dpdk-c.a
build/libndn-dpdk-c.a: build/build.ninja csrc/ndni/an.h csrc/ndni/enum.h csrc/iface/enum.h csrc/pcct/cs-enum.h
	ninja -C build

build/build.ninja: csrc/meson.build mk/meson.build
	bash -c 'source mk/cflags.sh; meson build'

build/cgodeps.done: build/build.ninja
	ninja -C build cgoflags cgostruct cgotest
	touch $@

csrc/meson.build mk/meson.build:
	mk/update-list.sh

.PHONY: tsc
tsc:
	node_modules/.bin/tsc

.PHONY: npm
npm: tsc
	mv $$(npm pack -s .) build/ndn-dpdk.tgz

.PHONY: cmds
cmds: build/bin/ndndpdk-afdump build/bin/ndndpdk-hrlog2histogram build/bin/ndnfw-dpdk build/bin/ndnping-dpdk

build/bin/%: cmd/%/* godeps
	GOBIN=$$(realpath build/bin) go install ./cmd/$*

install:
	mk/install.sh

uninstall:
	mk/uninstall.sh

doxygen:
	doxygen docs/Doxyfile 2>&1 | docs/filter-Doxygen-warning.awk 1>&2

mgmtspec: docs/mgmtspec.json

docs/mgmtspec.json:
	./node_modules/.bin/ts-node js/cmd/make-spec.ts >$@

.PHONY: docs
docs: doxygen mgmtspec

lint:
	mk/format-code.sh

test: godeps
	mk/gotest.sh

clean:
	awk '!(/node_modules/ || /\*\*/)' .dockerignore | xargs rm -rf
	awk 'BEGIN{FS="/"} $$1=="**"{print $$2}' .dockerignore | xargs -I{} -n1 find -name {} -delete
	go clean ./...
