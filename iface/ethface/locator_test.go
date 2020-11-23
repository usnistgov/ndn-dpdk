package ethface_test

import (
	"testing"

	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/iface/ethface"
)

func TestLocatorCoexist(t *testing.T) {
	assert, _ := makeAR(t)

	parse := func(j string) iface.Locator {
		var locw iface.LocatorWrapper
		fromJSON(j, &locw)
		return locw.Locator
	}
	coexist := func(a, b string) {
		assert.True(ethface.LocatorCanCoexist(parse(a), parse(b)))
	}
	conflict := func(a, b string) {
		assert.False(ethface.LocatorCanCoexist(parse(a), parse(b)))
	}

	// "ether" scheme
	const etherA = `,"local":"02:00:00:00:00:00","remote":"02:00:00:00:01:00"}`
	const etherB = `,"local":"02:00:00:00:00:00","remote":"02:00:00:00:01:01"}`
	conflict( // same MAC addresses and VLAN
		`{"scheme":"ether"`+etherA,
		`{"scheme":"ether"`+etherA)
	coexist( // different local
		`{"scheme":"ether","local":"02:00:00:00:00:00","remote":"02:00:00:00:01:00"}`,
		`{"scheme":"ether","local":"02:00:00:00:00:01","remote":"02:00:00:00:01:00"}`)
	coexist( // different remote
		`{"scheme":"ether"`+etherA,
		`{"scheme":"ether"`+etherB)
	coexist( // different VLAN
		`{"scheme":"ether"`+etherA,
		`{"scheme":"ether","vlan":2`+etherA)
	coexist( // unicast vs multicast
		`{"scheme":"ether","local":"02:00:00:00:00:00","remote":"02:00:00:00:01:00"}`,
		`{"scheme":"ether","local":"02:00:00:00:00:00","remote":"03:00:00:00:01:00"}`)
	conflict( // both multicast despite different addresses and VLAN
		`{"scheme":"ether","local":"02:00:00:00:00:00","remote":"03:00:00:00:01:00"}`,
		`{"scheme":"ether","local":"02:00:00:00:00:01","remote":"03:00:00:00:01:01","vlan":2}`)

	// "udpe" scheme
	const ipA = `,"localIP":"192.168.37.1","remoteIP":"192.168.37.2"`
	const ipB = `,"localIP":"192.168.37.1","remoteIP":"192.168.37.12"`
	conflict( // same IP addresses and ports, same MAC addresses
		`{"scheme":"udpe","localUDP":6363,"remoteUDP":6363`+ipA+etherA,
		`{"scheme":"udpe","localUDP":6363,"remoteUDP":6363`+ipA+etherA)
	conflict( // same IP addresses and ports, different MAC addresses
		`{"scheme":"udpe","localUDP":6363,"remoteUDP":6363`+ipA+etherA,
		`{"scheme":"udpe","localUDP":6363,"remoteUDP":6363`+ipA+etherB)
	coexist( // same IP addresses and ports, different VLAN
		`{"scheme":"udpe","localUDP":6363,"remoteUDP":6363`+ipA+etherA,
		`{"scheme":"udpe","localUDP":6363,"remoteUDP":6363,"vlan":2`+ipA+etherA)
	coexist( // different localIP
		`{"scheme":"udpe","localIP":"192.168.37.1","remoteIP":"192.168.37.2","localUDP":6363,"remoteUDP":6363`+etherA,
		`{"scheme":"udpe","localIP":"192.168.37.11","remoteIP":"192.168.37.2","localUDP":6363,"remoteUDP":6363`+etherA)
	coexist( // different remoteIP
		`{"scheme":"udpe","localUDP":6363,"remoteUDP":6363`+ipA+etherA,
		`{"scheme":"udpe","localUDP":6363,"remoteUDP":6363`+ipB+etherA)
	coexist( // different localUDP
		`{"scheme":"udpe","localUDP":6363,"remoteUDP":6363`+ipA+etherA,
		`{"scheme":"udpe","localUDP":16363,"remoteUDP":6363`+ipA+etherA)
	coexist( // different remoteUDP
		`{"scheme":"udpe","localUDP":6363,"remoteUDP":6363`+ipA+etherA,
		`{"scheme":"udpe","localUDP":6363,"remoteUDP":16363`+ipA+etherA)
	coexist( // IPv4 vs IPv6
		`{"scheme":"udpe","localUDP":6363,"remoteUDP":6363`+ipA+etherA,
		`{"scheme":"udpe","localIP":"fde0:fd0a:3557:a8c7:db87:639f:9bd2:0001","remoteIP":"fde0:fd0a:3557:a8c7:db87:639f:9bd2:0002",
		"localUDP":6363,"remoteUDP":6363`+etherA)

	// "vxlan" scheme
	const innerA = `,"innerLocal":"02:01:00:00:00:00","innerRemote":"02:01:00:00:01:00"`
	const innerB = `,"innerLocal":"02:01:00:00:00:00","innerRemote":"02:01:00:00:01:01"`
	conflict( // same IP addresses and ports, same outer MAC addresse
		`{"scheme":"vxlan","localUDP":4789,"remoteUDP":4789,"vxlan":1`+innerA+ipA+etherA,
		`{"scheme":"vxlan","localUDP":4789,"remoteUDP":4789,"vxlan":1`+innerA+ipA+etherA)
	conflict( // same IP addresses and ports, different outer MAC addresses
		`{"scheme":"vxlan","localUDP":4789,"remoteUDP":4789,"vxlan":1`+innerA+ipA+etherA,
		`{"scheme":"vxlan","localUDP":4789,"remoteUDP":4789,"vxlan":1`+innerA+ipA+etherB)
	coexist( // same IP addresses and ports, different outer VLAN
		`{"scheme":"vxlan","localUDP":4789,"remoteUDP":4789,"vxlan":1`+innerA+ipA+etherA,
		`{"scheme":"vxlan","localUDP":4789,"remoteUDP":4789,"vxlan":1,"vlan":2`+innerA+ipA+etherA)
	coexist( // different IP addresses
		`{"scheme":"vxlan","localUDP":4789,"remoteUDP":4789,"vxlan":1`+innerA+ipA+etherA,
		`{"scheme":"vxlan","localUDP":4789,"remoteUDP":4789,"vxlan":1`+innerA+ipB+etherA)
	conflict( // same localUDP, different remoteUDP
		`{"scheme":"vxlan","localUDP":4789,"remoteUDP":4789,"vxlan":1`+innerA+ipA+etherA,
		`{"scheme":"vxlan","localUDP":4789,"remoteUDP":14789,"vxlan":1`+innerA+ipA+etherA)
	conflict( // different localUDP, same remoteUDP
		`{"scheme":"vxlan","localUDP":4789,"remoteUDP":4789,"vxlan":1`+innerA+ipA+etherA,
		`{"scheme":"vxlan","localUDP":14789,"remoteUDP":4789,"vxlan":1`+innerA+ipA+etherA)
	coexist( // different localUDP, different remoteUDP
		`{"scheme":"vxlan","localUDP":4789,"remoteUDP":4789,"vxlan":1`+innerA+ipA+etherA,
		`{"scheme":"vxlan","localUDP":14789,"remoteUDP":14789,"vxlan":1`+innerA+ipA+etherA)
	coexist( // same IP addresses and ports, but different VNI
		`{"scheme":"vxlan","localUDP":4789,"remoteUDP":4789,"vxlan":0`+innerA+ipA+etherA,
		`{"scheme":"vxlan","localUDP":4789,"remoteUDP":4789,"vxlan":1`+innerA+ipA+etherA)
	coexist( // same IP addresses and ports, but different inner MAC addresses
		`{"scheme":"vxlan","localUDP":4789,"remoteUDP":4789,"vxlan":1`+innerA+ipA+etherA,
		`{"scheme":"vxlan","localUDP":4789,"remoteUDP":4789,"vxlan":1`+innerB+ipA+etherA)

	// mixed schemes
	coexist( // "ether" with "udpe"
		`{"scheme":"ether"`+etherA,
		`{"scheme":"udpe","localUDP":6363,"remoteUDP":6363`+ipA+etherA)
	coexist( // "ether" with "vxlan"
		`{"scheme":"ether"`+etherA,
		`{"scheme":"vxlan","localUDP":4789,"remoteUDP":4789,"vxlan":1`+innerA+ipA+etherA)
	conflict( // "udp" with "vxlan", same localUDP
		`{"scheme":"udpe","localUDP":4444,"remoteUDP":4444`+ipA+etherA,
		`{"scheme":"vxlan","localUDP":4444,"remoteUDP":14444,"vxlan":1`+innerA+ipA+etherA)
	conflict( // "udp" with "vxlan", same remoteUDP
		`{"scheme":"udpe","localUDP":4444,"remoteUDP":4444`+ipA+etherA,
		`{"scheme":"vxlan","localUDP":14444,"remoteUDP":4444,"vxlan":1`+innerA+ipA+etherA)
	coexist( // "udp" with "vxlan", different ports
		`{"scheme":"udpe","localUDP":6363,"remoteUDP":6363`+ipA+etherA,
		`{"scheme":"vxlan","localUDP":4789,"remoteUDP":4789,"vxlan":1`+innerA+ipA+etherA)
}
