.PHONY: all
all: gopkg cmds npm

.PHONY: gopkg
gopkg: godeps
	mk/go.sh build -v ./...

.PHONY: godeps
godeps: build/libndn-dpdk-c.a build/cgodeps.done build/bpf.done

csrc/meson.build mk/meson.build:
	mk/update-list.sh

build/build.ninja: csrc/meson.build mk/meson.build
	bash -c 'source mk/cflags.sh; meson setup $$MESONFLAGS build'

csrc/core/rttest-enum.h: core/rttest/rttest.go
	mk/go.sh generate ./$(<D)

csrc/dpdk/bdev-enum.h: dpdk/bdev/write-mode.go
	mk/go.sh generate ./$(<D)

csrc/dpdk/thread-enum.h: dpdk/ealthread/ctrl.go
	mk/go.sh generate ./$(<D)

csrc/fileserver/enum.h csrc/fileserver/an.h: app/fileserver/config.go ndn/rdr/ndn6file/*.go
	mk/go.sh generate ./$(<D)

csrc/fib/enum.h: container/fib/fibdef/enum.go
	mk/go.sh generate ./$(<D)

csrc/ndni/enum.h csrc/ndni/an.h: ndni/enum.go ndn/an/*.go
	mk/go.sh generate ./$(<D)

csrc/iface/enum.h: iface/enum.go
	mk/go.sh generate ./$(<D)

csrc/pcct/cs-enum.h: container/cs/enum.go
	mk/go.sh generate ./$(<D)

csrc/pdump/enum.h: app/pdump/enum.go
	mk/go.sh generate ./$(<D)

csrc/tgconsumer/enum.h: app/tgconsumer/config.go
	mk/go.sh generate ./$(<D)

csrc/tgproducer/enum.h: app/tgproducer/config.go
	mk/go.sh generate ./$(<D)

.PHONY: build/libndn-dpdk-c.a
build/libndn-dpdk-c.a: build/build.ninja csrc/core/rttest-enum.h csrc/dpdk/bdev-enum.h csrc/dpdk/thread-enum.h csrc/fib/enum.h csrc/fileserver/an.h csrc/fileserver/enum.h csrc/ndni/an.h csrc/ndni/enum.h csrc/iface/enum.h csrc/pcct/cs-enum.h csrc/pdump/enum.h csrc/tgconsumer/enum.h csrc/tgproducer/enum.h
	meson compile -C build

build/cgodeps.done: build/build.ninja
	meson compile -C build cgoflags cgostruct cgotest schema
	touch $@

build/bpf.done: build/build.ninja bpf/**/*.* csrc/strategyapi/* csrc/fib/enum.h csrc/ndni/an.h csrc/pcct/pit-const.h
	meson compile -C build bpf
	touch $@

.PHONY: cmds
cmds: build/share/bash_autocomplete build/bin/ndndpdk-ctrl build/bin/ndndpdk-godemo build/bin/ndndpdk-hrlog2histogram build/bin/ndndpdk-jrproxy build/bin/ndndpdk-svc build/bin/ndndpdk-upf

build/bin/%: cmd/%/* godeps
	GOBIN=$$(realpath build/bin) mk/go.sh install ./cmd/$*

build/share/bash_autocomplete: go.mod
	mkdir -p $(@D)
	install -m0644 $$(go env GOMODCACHE)/github.com/urfave/cli/v2@$$(awk '$$1=="github.com/urfave/cli/v2" { print $$2 }' go.mod)/autocomplete/bash_autocomplete $@

.PHONY: npm
npm: build/share/ndn-dpdk/ndn-dpdk.npm.tgz

build/share/ndn-dpdk/ndn-dpdk.npm.tgz:
	$$(corepack pnpm bin)/tsc
	mkdir -p $(@D)
	mv $$(corepack pnpm pack --json . | jq -r .filename) $@

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

.PHONY: coverage
coverage:
	ninja -C build coverage-html

.PHONY: coverage-clean
coverage-clean:
	find build -name '*.gcda' -delete

.PHONY: clean
clean:
	awk '!(/node_modules/ || /pnpm-lock/ || /\*/)' .dockerignore | xargs rm -rf
	awk '/\*/' .dockerignore | xargs -I{} -n1 find -wholename ./{} -delete
	mk/go.sh clean ./...
