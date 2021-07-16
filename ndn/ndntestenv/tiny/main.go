// Command tiny ensures part of NDNgo library is compatible with TinyGo compiler.
// Supported packages should be written as imports.
package main

import (
	_ "github.com/usnistgov/ndn-dpdk/ndn"
	_ "github.com/usnistgov/ndn-dpdk/ndn/an"
	_ "github.com/usnistgov/ndn-dpdk/ndn/keychain"
	_ "github.com/usnistgov/ndn-dpdk/ndn/tlv"
)

func main() {
}
