// Command tiny verifies that part of NDNgo library is compatible with TinyGo compiler.
// Supported packages should be written as imports.
package main

import (
	_ "github.com/usnistgov/ndn-dpdk/ndn"
	_ "github.com/usnistgov/ndn-dpdk/ndn/an"
	_ "github.com/usnistgov/ndn-dpdk/ndn/endpoint"
	_ "github.com/usnistgov/ndn-dpdk/ndn/keychain"
	_ "github.com/usnistgov/ndn-dpdk/ndn/l3"
	_ "github.com/usnistgov/ndn-dpdk/ndn/mgmt"
	_ "github.com/usnistgov/ndn-dpdk/ndn/rdr"
	_ "github.com/usnistgov/ndn-dpdk/ndn/rdr/ndn6file"
	_ "github.com/usnistgov/ndn-dpdk/ndn/segmented"
	_ "github.com/usnistgov/ndn-dpdk/ndn/tlv"
	_ "github.com/usnistgov/ndn-dpdk/ndn/wasmtransport"
)

func main() {
}
