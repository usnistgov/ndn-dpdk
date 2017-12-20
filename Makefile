all: go-dpdk go-ndn go-face

cmd-%: cmd/%/* go-dpdk go-ndn go-face
	go install ./cmd/$*

go-dpdk: dpdk/*.go
	go build ./dpdk

build-c/libndn-dpdk-dpdk.a: dpdk/*.c
	./build-c.sh dpdk

go-ndn: go-dpdk ndn/*.go ndn/error.go ndn/tlv-type.go build-c/libndn-dpdk-dpdk.a
	go build ./ndn

ndn/error.go ndn/error.h: ndn/make-error.sh ndn/error.tsv
	ndn/make-error.sh

ndn/tlv-type.go ndn/tlv-type.h: ndn/make-tlv-type.sh ndn/tlv-type.tsv
	ndn/make-tlv-type.sh

build-c/libndn-dpdk-ndn.a: ndn/*.c ndn/error.h ndn/tlv-type.h
	./build-c.sh ndn

go-face: go-ndn face/*.go build-c/libndn-dpdk-dpdk.a build-c/libndn-dpdk-ndn.a
	go build ./face

test:
	./gotest.sh dpdk
	./gotest.sh ndn
	./gotest.sh face
	integ/run.sh

clean:
	rm -rf build-c ndn/error.go ndn/error.h ndn/tlv-type.go ndn/tlv-type.h
	go clean ./...

doxygen:
	cd docs && doxygen Doxyfile 2>&1 | ./filter-Doxygen-warning.awk 1>&2

dochttp: doxygen
	cd docs/html && python3 -m http.server 2>/dev/null &