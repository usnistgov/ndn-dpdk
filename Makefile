export CGO_CFLAGS_ALLOW := '.*'
ifeq ($(origin CC),default)
	CC = gcc
endif
export CC

.PHONY: all
all: gopkg cmds npm

.PHONY: gopkg
gopkg: godeps
	go build -v ./...

.PHONY: godeps
godeps: build/libndn-dpdk-c.a build/cgodeps.done build/strategy.done

csrc/fib/enum.h: container/fib/fibdef/enum.go
	mk/gogenerate.sh ./$(<D)

csrc/ndni/enum.h csrc/ndni/an.h: ndni/enum.go ndn/an/*.go
	mk/gogenerate.sh ./$(<D)

csrc/iface/enum.h: iface/enum.go
	mk/gogenerate.sh ./$(<D)

csrc/pcct/cs-enum.h: container/cs/enum.go
	mk/gogenerate.sh ./$(<D)

build/strategy.done: strategy/*.c csrc/strategyapi/* csrc/fib/enum.h
	strategy/compile.sh

.PHONY: build/libndn-dpdk-c.a
build/libndn-dpdk-c.a: build/build.ninja csrc/fib/enum.h csrc/ndni/an.h csrc/ndni/enum.h csrc/iface/enum.h csrc/pcct/cs-enum.h
	ninja -C build

build/build.ninja: csrc/meson.build mk/meson.build
	bash -c 'source mk/cflags.sh; meson build'

build/cgodeps.done: build/build.ninja
	ninja -C build cgoflags cgostruct cgotest schema
	touch $@

csrc/meson.build mk/meson.build:
	mk/update-list.sh

.PHONY: cmds
cmds: build/bin/ndndpdk-ctrl build/bin/ndndpdk-godemo build/bin/ndndpdk-hrlog2histogram build/bin/ndndpdk-jrproxy build/bin/ndndpdk-svc

build/bin/%: cmd/%/* godeps
	GOBIN=$$(realpath build/bin) go install "-ldflags=$$(mk/version/ldflags.sh)" ./cmd/$*

.PHONY: npm
npm: build/share/ndn-dpdk/ndn-dpdk.npm.tgz

build/share/ndn-dpdk/ndn-dpdk.npm.tgz:
	node_modules/.bin/tsc
	mv $$(npm pack -s .) $@

.PHONY: install
install:
	mk/install.sh

.PHONY: uninstall
uninstall:
	mk/uninstall.sh

.PHONY: doxygen
doxygen:
	doxygen docs/Doxyfile 2>&1 | docs/filter-Doxygen-warning.awk 1>&2

.PHONY: lint
lint:
	mk/format-code.sh

.PHONY: test
test: godeps
	mk/gotest.sh

.PHONY: clean
clean:
	awk '!(/node_modules/ || /\*/)' .dockerignore | xargs rm -rf
	awk '/\*/' .dockerignore | xargs -I{} -n1 find -wholename ./{} -delete
	go clean -cache ./...
