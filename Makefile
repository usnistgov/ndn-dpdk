export CGO_CFLAGS_ALLOW := '.*'
ifeq ($(origin CC),default)
	CC = gcc-7
endif
export CC

all: gopkg tsc cmds

gopkg: godeps
	go build -v ./...

godeps: app/version/version.go
	rake cgostruct cgoflags cbuilds strategies

.PHONY: app/version/version.go
app/version/version.go:
	app/version/make-version.sh

.PHONY: tsc
tsc: ndn/tlv-type.ts
	node_modules/.bin/tsc

ndn/tlv-type.ts: ndn/tlv-type.tsv
	rake ndn/tlv-type.h

cmds: cmd-ndnfw-dpdk cmd-ndnping-dpdk

cmd-%: cmd/%/* godeps
	go install ./cmd/$*

goget:
	go get -d -t ./...

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
