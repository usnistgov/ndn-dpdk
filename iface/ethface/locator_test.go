package ethface_test

import (
	"crypto/rand"
	"net"
	"testing"

	"slices"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"
	"github.com/usnistgov/ndn-dpdk/iface/ethport"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/an"
	"github.com/usnistgov/ndn-dpdk/ndn/ndnlayer"
	"github.com/usnistgov/ndn-dpdk/ndn/tlv"
)

func TestLocatorCoexist(t *testing.T) {
	assert, _ := makeAR(t)
	coexist := func(a, b string) {
		assert.NoError(ethport.CheckLocatorCoexist(parseLocator(a), parseLocator(b)))
	}
	conflict := func(a, b string) {
		assert.Error(ethport.CheckLocatorCoexist(parseLocator(a), parseLocator(b)))
	}

	// "passthru" scheme
	const passthru = `{"scheme":"passthru","local":"02:00:00:00:00:00"}`
	conflict(passthru, passthru)

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
	conflict( // same IP addresses, same outer MAC addresse
		`{"scheme":"vxlan","vxlan":1`+innerA+ipA+etherA,
		`{"scheme":"vxlan","vxlan":1`+innerA+ipA+etherA)
	conflict( // same IP addresses, different outer MAC addresses
		`{"scheme":"vxlan","vxlan":1`+innerA+ipA+etherA,
		`{"scheme":"vxlan","vxlan":1`+innerA+ipA+etherB)
	coexist( // same IP addresses, different outer VLAN
		`{"scheme":"vxlan","vxlan":1`+innerA+ipA+etherA,
		`{"scheme":"vxlan","vxlan":1,"vlan":2`+innerA+ipA+etherA)
	coexist( // different IP addresses
		`{"scheme":"vxlan","vxlan":1`+innerA+ipA+etherA,
		`{"scheme":"vxlan","vxlan":1`+innerA+ipB+etherA)
	coexist( // same IP addresses, different VNI
		`{"scheme":"vxlan","vxlan":0`+innerA+ipA+etherA,
		`{"scheme":"vxlan","vxlan":1`+innerA+ipA+etherA)
	coexist( // same IP addresses, different inner MAC addresses
		`{"scheme":"vxlan","vxlan":1`+innerA+ipA+etherA,
		`{"scheme":"vxlan","vxlan":1`+innerB+ipA+etherA)

	const innerIP = `,"innerLocalIP":"192.168.60.3","innerRemoteIP":"192.168.60.4"`
	conflict( // same ulTEID
		`{"scheme":"gtp","ulTEID":268435464,"ulQFI":1,"dlTEID":536870920,"dlQFI":1`+innerIP+ipA+etherA,
		`{"scheme":"gtp","ulTEID":268435464,"ulQFI":2,"dlTEID":536870921,"dlQFI":2`+innerIP+ipA+etherA,
	)
	conflict( // same dlTEID
		`{"scheme":"gtp","ulTEID":268435464,"ulQFI":1,"dlTEID":536870920,"dlQFI":1`+innerIP+ipA+etherA,
		`{"scheme":"gtp","ulTEID":268435465,"ulQFI":2,"dlTEID":536870920,"dlQFI":2`+innerIP+ipA+etherA,
	)
	coexist( // different ulTEID and dlTEID
		`{"scheme":"gtp","ulTEID":268435464,"ulQFI":1,"dlTEID":536870920,"dlQFI":1`+innerIP+ipA+etherA,
		`{"scheme":"gtp","ulTEID":268435465,"ulQFI":1,"dlTEID":536870921,"dlQFI":1`+innerIP+ipA+etherA,
	)

	// mixed schemes
	coexist( // "passthru" with "ether"
		passthru,
		`{"scheme":"ether"`+etherA,
	)
	coexist( // "ether" with "udpe"
		`{"scheme":"ether"`+etherA,
		`{"scheme":"udpe","localUDP":6363,"remoteUDP":6363`+ipA+etherA)
	coexist( // "ether" with "vxlan"
		`{"scheme":"ether"`+etherA,
		`{"scheme":"vxlan","vxlan":1`+innerA+ipA+etherA)
	conflict( // "udpe" with "vxlan", same localUDP
		`{"scheme":"udpe","localUDP":4789,"remoteUDP":4444`+ipA+etherA,
		`{"scheme":"vxlan","vxlan":1`+innerA+ipA+etherA)
	conflict( // "udpe" with "vxlan", same remoteUDP
		`{"scheme":"udpe","localUDP":4444,"remoteUDP":4789`+ipA+etherA,
		`{"scheme":"vxlan","vxlan":1`+innerA+ipA+etherA)
	coexist( // "udpe" with "vxlan", different ports
		`{"scheme":"udpe","localUDP":6363,"remoteUDP":6363`+ipA+etherA,
		`{"scheme":"vxlan","vxlan":1`+innerA+ipA+etherA)
	conflict( // "udpe" with "gtp", same localUDP and remoteUDP
		`{"scheme":"udpe","localUDP":2152,"remoteUDP":2152`+ipA+etherA,
		`{"scheme":"gtp","ulTEID":268435464,"ulQFI":1,"dlTEID": 536870920,"dlQFI":1`+innerIP+ipA+etherA)
}

func TestLocatorRxMatch(t *testing.T) {
	assert, _ := makeAR(t)

	matchers := map[string]ethport.RxMatch{}
	addMatcher := func(key string, locator string) {
		matchers[key] = ethport.NewRxMatch(parseLocator(locator))
	}
	addMatcher("ether-unicast", `{
		"scheme": "ether",
		"local": "02:00:00:00:00:01",
		"remote": "02:00:00:00:00:02"
	}`)
	addMatcher("ether-unicast-vlan", `{
		"scheme": "ether",
		"local": "02:00:00:00:00:01",
		"remote": "02:00:00:00:00:02",
		"vlan": 3
	}`)
	addMatcher("ether-multicast", `{
		"scheme": "ether",
		"local": "02:00:00:00:00:01",
		"remote": "01:00:5E:00:17:AA"
	}`)
	addMatcher("ether-multicast-vlan", `{
		"scheme": "ether",
		"local": "02:00:00:00:00:01",
		"remote": "01:00:5E:00:17:AA",
		"vlan": 3
	}`)
	addMatcher("udp4", `{
		"scheme": "udpe",
		"local": "02:00:00:00:00:01",
		"remote": "02:00:00:00:00:02",
		"localIP": "192.168.37.1",
		"remoteIP": "192.168.37.2",
		"localUDP": 6363,
		"remoteUDP": 16363
	}`)
	addMatcher("udp6", `{
		"scheme": "udpe",
		"local": "02:00:00:00:00:01",
		"remote": "02:00:00:00:00:02",
		"localIP": "fde0:fd0a:3557:a8c7:db87:639f:9bd2:0001",
		"remoteIP": "fde0:fd0a:3557:a8c7:db87:639f:9bd2:0002",
		"localUDP": 6363,
		"remoteUDP": 16363
	}`)
	addMatcher("vxlan0", `{
		"scheme": "vxlan",
		"local": "02:00:00:00:00:01",
		"remote": "02:00:00:00:00:02",
		"localIP": "fde0:fd0a:3557:a8c7:db87:639f:9bd2:0001",
		"remoteIP": "fde0:fd0a:3557:a8c7:db87:639f:9bd2:0002",
		"vxlan": 0,
		"innerLocal": "02:00:00:00:00:03",
		"innerRemote": "02:00:00:00:00:04"
	}`)
	addMatcher("vxlan1", `{
		"scheme": "vxlan",
		"local": "02:00:00:00:00:01",
		"remote": "02:00:00:00:00:02",
		"localIP": "fde0:fd0a:3557:a8c7:db87:639f:9bd2:0001",
		"remoteIP": "fde0:fd0a:3557:a8c7:db87:639f:9bd2:0002",
		"vxlan": 1,
		"innerLocal": "02:00:00:00:00:03",
		"innerRemote": "02:00:00:00:00:04"
	}`)
	addMatcher("gtp4", `{
		"scheme": "gtp",
		"local": "02:00:00:00:00:01",
		"remote": "02:00:00:00:00:02",
		"localIP": "192.168.37.1",
		"remoteIP": "192.168.37.2",
		"ulTEID": 268435464,
		"ulQFI": 1,
		"dlTEID": 536870920,
		"dlQFI": 11,
		"innerLocalIP": "192.168.60.3",
		"innerRemoteIP": "192.168.60.4"
	}`)
	addMatcher("gtp6", `{
		"scheme": "gtp",
		"local": "02:00:00:00:00:01",
		"remote": "02:00:00:00:00:02",
		"vlan": 3,
		"localIP": "fde0:fd0a:3557:a8c7:db87:639f:9bd2:0001",
		"remoteIP": "fde0:fd0a:3557:a8c7:db87:639f:9bd2:0002",
		"ulTEID": 268435464,
		"ulQFI": 1,
		"dlTEID": 536870920,
		"dlQFI": 11,
		"innerLocalIP": "192.168.60.3",
		"innerRemoteIP": "192.168.60.4"
	}`)

	payload := make(gopacket.Payload, 200)
	rand.Read([]byte(payload))
	onlyMatch := func(matcherKey string, headers ...gopacket.SerializableLayer) {
		pkt := pktmbufFromLayers(append(slices.Clone(headers), payload)...)
		defer pkt.Close()

		for key, m := range matchers {
			if key == matcherKey {
				assert.True(m.Match(pkt))
				assert.Equal([]byte(payload), pkt.SegmentBytes()[0][m.HdrLen():])
			} else {
				assert.False(m.Match(pkt))
			}
		}
	}

	mac0 := net.HardwareAddr{0x02, 0x00, 0x00, 0x00, 0x00, 0x00}
	mac1 := net.HardwareAddr{0x02, 0x00, 0x00, 0x00, 0x00, 0x01}
	mac2 := net.HardwareAddr{0x02, 0x00, 0x00, 0x00, 0x00, 0x02}
	mac3 := net.HardwareAddr{0x02, 0x00, 0x00, 0x00, 0x00, 0x03}
	mac4 := net.HardwareAddr{0x02, 0x00, 0x00, 0x00, 0x00, 0x04}
	macM := net.HardwareAddr{0x01, 0x00, 0x5E, 0x00, 0x17, 0xAA}
	ip40 := net.ParseIP("192.168.37.0")
	ip41 := net.ParseIP("192.168.37.1")
	ip42 := net.ParseIP("192.168.37.2")
	ip43 := net.ParseIP("192.168.60.3")
	ip44 := net.ParseIP("192.168.60.4")
	ip60 := net.ParseIP("fde0:fd0a:3557:a8c7:db87:639f:9bd2:0000")
	ip61 := net.ParseIP("fde0:fd0a:3557:a8c7:db87:639f:9bd2:0001")
	ip62 := net.ParseIP("fde0:fd0a:3557:a8c7:db87:639f:9bd2:0002")

	onlyMatch("ether-unicast",
		&layers.Ethernet{SrcMAC: mac2, DstMAC: mac1, EthernetType: an.EtherTypeNDN},
	)
	onlyMatch("", // wrong EtherType
		&layers.Ethernet{SrcMAC: mac2, DstMAC: mac1, EthernetType: layers.EthernetTypeDot1Q},
	)
	onlyMatch("", // wrong SrcMAC
		&layers.Ethernet{SrcMAC: mac0, DstMAC: mac1, EthernetType: layers.EthernetTypeDot1Q},
	)
	onlyMatch("", // wrong DstMAC
		&layers.Ethernet{SrcMAC: mac2, DstMAC: mac0, EthernetType: layers.EthernetTypeDot1Q},
	)
	onlyMatch("ether-unicast-vlan",
		&layers.Ethernet{SrcMAC: mac2, DstMAC: mac1, EthernetType: layers.EthernetTypeDot1Q},
		&layers.Dot1Q{VLANIdentifier: 3, Type: an.EtherTypeNDN},
	)
	onlyMatch("", // wrong VLAN
		&layers.Ethernet{SrcMAC: mac2, DstMAC: mac1, EthernetType: layers.EthernetTypeDot1Q},
		&layers.Dot1Q{VLANIdentifier: 4, Type: an.EtherTypeNDN},
	)
	onlyMatch("ether-multicast",
		&layers.Ethernet{SrcMAC: mac2, DstMAC: macM, EthernetType: an.EtherTypeNDN},
	)
	onlyMatch("ether-multicast-vlan",
		&layers.Ethernet{SrcMAC: mac2, DstMAC: macM, EthernetType: layers.EthernetTypeDot1Q},
		&layers.Dot1Q{VLANIdentifier: 3, Type: an.EtherTypeNDN},
	)

	onlyMatch("udp4",
		&layers.Ethernet{SrcMAC: mac2, DstMAC: mac1, EthernetType: layers.EthernetTypeIPv4},
		&layers.IPv4{Version: 4, TTL: 64, Protocol: layers.IPProtocolUDP, SrcIP: ip42, DstIP: ip41},
		&layers.UDP{SrcPort: 16363, DstPort: 6363},
	)
	onlyMatch("", // wrong SrcIP
		&layers.Ethernet{SrcMAC: mac2, DstMAC: mac1, EthernetType: layers.EthernetTypeIPv4},
		&layers.IPv4{Version: 4, TTL: 64, Protocol: layers.IPProtocolUDP, SrcIP: ip40, DstIP: ip41},
		&layers.UDP{SrcPort: 16363, DstPort: 6363},
	)
	onlyMatch("", // wrong DstIP
		&layers.Ethernet{SrcMAC: mac2, DstMAC: mac1, EthernetType: layers.EthernetTypeIPv4},
		&layers.IPv4{Version: 4, TTL: 64, Protocol: layers.IPProtocolUDP, SrcIP: ip42, DstIP: ip40},
		&layers.UDP{SrcPort: 16363, DstPort: 6363},
	)
	onlyMatch("", // wrong SrcPort
		&layers.Ethernet{SrcMAC: mac2, DstMAC: mac1, EthernetType: layers.EthernetTypeIPv4},
		&layers.IPv4{Version: 4, TTL: 64, Protocol: layers.IPProtocolUDP, SrcIP: ip42, DstIP: ip41},
		&layers.UDP{SrcPort: 26363, DstPort: 6363},
	)
	onlyMatch("", // wrong DstPort
		&layers.Ethernet{SrcMAC: mac2, DstMAC: mac1, EthernetType: layers.EthernetTypeIPv4},
		&layers.IPv4{Version: 4, TTL: 64, Protocol: layers.IPProtocolUDP, SrcIP: ip42, DstIP: ip41},
		&layers.UDP{SrcPort: 16363, DstPort: 26363},
	)

	onlyMatch("udp6",
		&layers.Ethernet{SrcMAC: mac2, DstMAC: mac1, EthernetType: layers.EthernetTypeIPv6},
		&layers.IPv6{Version: 6, NextHeader: layers.IPProtocolUDP, SrcIP: ip62, DstIP: ip61},
		&layers.UDP{SrcPort: 16363, DstPort: 6363},
	)
	onlyMatch("", // wrong SrcIP
		&layers.Ethernet{SrcMAC: mac2, DstMAC: mac1, EthernetType: layers.EthernetTypeIPv6},
		&layers.IPv6{Version: 6, NextHeader: layers.IPProtocolUDP, SrcIP: ip60, DstIP: ip61},
		&layers.UDP{SrcPort: 16363, DstPort: 6363},
	)
	onlyMatch("", // wrong DstIP
		&layers.Ethernet{SrcMAC: mac2, DstMAC: mac1, EthernetType: layers.EthernetTypeIPv6},
		&layers.IPv6{Version: 6, NextHeader: layers.IPProtocolUDP, SrcIP: ip62, DstIP: ip60},
		&layers.UDP{SrcPort: 16363, DstPort: 6363},
	)
	onlyMatch("", // wrong SrcPort
		&layers.Ethernet{SrcMAC: mac2, DstMAC: mac1, EthernetType: layers.EthernetTypeIPv6},
		&layers.IPv6{Version: 6, NextHeader: layers.IPProtocolUDP, SrcIP: ip62, DstIP: ip61},
		&layers.UDP{SrcPort: 26363, DstPort: 6363},
	)
	onlyMatch("", // wrong DstPort
		&layers.Ethernet{SrcMAC: mac2, DstMAC: mac1, EthernetType: layers.EthernetTypeIPv6},
		&layers.IPv6{Version: 6, NextHeader: layers.IPProtocolUDP, SrcIP: ip62, DstIP: ip61},
		&layers.UDP{SrcPort: 16363, DstPort: 26363},
	)

	onlyMatch("vxlan0",
		&layers.Ethernet{SrcMAC: mac2, DstMAC: mac1, EthernetType: layers.EthernetTypeIPv6},
		&layers.IPv6{Version: 6, NextHeader: layers.IPProtocolUDP, SrcIP: ip62, DstIP: ip61},
		&layers.UDP{SrcPort: 65000, DstPort: 4789},
		&layers.VXLAN{VNI: 0},
		&layers.Ethernet{SrcMAC: mac4, DstMAC: mac3, EthernetType: an.EtherTypeNDN},
	)
	onlyMatch("vxlan1",
		&layers.Ethernet{SrcMAC: mac2, DstMAC: mac1, EthernetType: layers.EthernetTypeIPv6},
		&layers.IPv6{Version: 6, NextHeader: layers.IPProtocolUDP, SrcIP: ip62, DstIP: ip61},
		&layers.UDP{SrcPort: 65000, DstPort: 4789},
		&layers.VXLAN{VNI: 1},
		&layers.Ethernet{SrcMAC: mac4, DstMAC: mac3, EthernetType: an.EtherTypeNDN},
	)
	onlyMatch("", // wrong inner SrcMAC
		&layers.Ethernet{SrcMAC: mac2, DstMAC: mac1, EthernetType: layers.EthernetTypeIPv6},
		&layers.IPv6{Version: 6, NextHeader: layers.IPProtocolUDP, SrcIP: ip62, DstIP: ip61},
		&layers.UDP{SrcPort: 65000, DstPort: 4789},
		&layers.VXLAN{VNI: 1},
		&layers.Ethernet{SrcMAC: mac0, DstMAC: mac3, EthernetType: an.EtherTypeNDN},
	)
	onlyMatch("", // wrong inner DstMAC
		&layers.Ethernet{SrcMAC: mac2, DstMAC: mac1, EthernetType: layers.EthernetTypeIPv6},
		&layers.IPv6{Version: 6, NextHeader: layers.IPProtocolUDP, SrcIP: ip62, DstIP: ip61},
		&layers.UDP{SrcPort: 65000, DstPort: 4789},
		&layers.VXLAN{VNI: 1},
		&layers.Ethernet{SrcMAC: mac4, DstMAC: mac0, EthernetType: an.EtherTypeNDN},
	)
	onlyMatch("", // wrong inner EtherType
		&layers.Ethernet{SrcMAC: mac2, DstMAC: mac1, EthernetType: layers.EthernetTypeIPv6},
		&layers.IPv6{Version: 6, NextHeader: layers.IPProtocolUDP, SrcIP: ip62, DstIP: ip61},
		&layers.UDP{SrcPort: 65000, DstPort: 4789},
		&layers.VXLAN{VNI: 1},
		&layers.Ethernet{SrcMAC: mac4, DstMAC: mac3, EthernetType: layers.EthernetTypePPP},
	)

	onlyMatch("gtp4",
		&layers.Ethernet{SrcMAC: mac2, DstMAC: mac1, EthernetType: layers.EthernetTypeIPv4},
		&layers.IPv4{Version: 4, TTL: 64, Protocol: layers.IPProtocolUDP, SrcIP: ip42, DstIP: ip41},
		&layers.UDP{SrcPort: 2152, DstPort: 2152},
		makeGTPv1U(0x10000008, 1, 1),
		&layers.IPv4{Version: 4, TTL: 64, Protocol: layers.IPProtocolUDP, SrcIP: ip44, DstIP: ip43},
		&layers.UDP{SrcPort: 6363, DstPort: 6363},
	)
	onlyMatch("", // wrong TEID
		&layers.Ethernet{SrcMAC: mac2, DstMAC: mac1, EthernetType: layers.EthernetTypeIPv4},
		&layers.IPv4{Version: 4, TTL: 64, Protocol: layers.IPProtocolUDP, SrcIP: ip42, DstIP: ip41},
		&layers.UDP{SrcPort: 2152, DstPort: 2152},
		makeGTPv1U(0x10000009, 1, 1),
		&layers.IPv4{Version: 4, TTL: 64, Protocol: layers.IPProtocolUDP, SrcIP: ip44, DstIP: ip43},
		&layers.UDP{SrcPort: 6363, DstPort: 6363},
	)
	onlyMatch("", // wrong TEID (using downlink TEID in uplink packet)
		&layers.Ethernet{SrcMAC: mac2, DstMAC: mac1, EthernetType: layers.EthernetTypeIPv4},
		&layers.IPv4{Version: 4, TTL: 64, Protocol: layers.IPProtocolUDP, SrcIP: ip42, DstIP: ip41},
		&layers.UDP{SrcPort: 2152, DstPort: 2152},
		makeGTPv1U(0x20000008, 1, 1),
		&layers.IPv4{Version: 4, TTL: 64, Protocol: layers.IPProtocolUDP, SrcIP: ip44, DstIP: ip43},
		&layers.UDP{SrcPort: 6363, DstPort: 6363},
	)
	onlyMatch("", // wrong QFI
		&layers.Ethernet{SrcMAC: mac2, DstMAC: mac1, EthernetType: layers.EthernetTypeIPv4},
		&layers.IPv4{Version: 4, TTL: 64, Protocol: layers.IPProtocolUDP, SrcIP: ip42, DstIP: ip41},
		&layers.UDP{SrcPort: 2152, DstPort: 2152},
		makeGTPv1U(0x10000008, 1, 11),
		&layers.IPv4{Version: 4, TTL: 64, Protocol: layers.IPProtocolUDP, SrcIP: ip44, DstIP: ip43},
		&layers.UDP{SrcPort: 6363, DstPort: 6363},
	)
	onlyMatch("", // wrong PDU session container type
		&layers.Ethernet{SrcMAC: mac2, DstMAC: mac1, EthernetType: layers.EthernetTypeIPv4},
		&layers.IPv4{Version: 4, TTL: 64, Protocol: layers.IPProtocolUDP, SrcIP: ip42, DstIP: ip41},
		&layers.UDP{SrcPort: 2152, DstPort: 2152},
		makeGTPv1U(0x10000008, 0, 1),
		&layers.IPv4{Version: 4, TTL: 64, Protocol: layers.IPProtocolUDP, SrcIP: ip44, DstIP: ip43},
		&layers.UDP{SrcPort: 6363, DstPort: 6363},
	)
	onlyMatch("", // missing PDU session container
		&layers.Ethernet{SrcMAC: mac2, DstMAC: mac1, EthernetType: layers.EthernetTypeIPv4},
		&layers.IPv4{Version: 4, TTL: 64, Protocol: layers.IPProtocolUDP, SrcIP: ip42, DstIP: ip41},
		&layers.UDP{SrcPort: 2152, DstPort: 2152},
		&layers.GTPv1U{Version: 1, ProtocolType: 1, MessageType: 0xFF},
		&layers.IPv4{Version: 4, TTL: 64, Protocol: layers.IPProtocolUDP, SrcIP: ip44, DstIP: ip43},
		&layers.UDP{SrcPort: 6363, DstPort: 6363},
	)

	onlyMatch("gtp6",
		&layers.Ethernet{SrcMAC: mac2, DstMAC: mac1, EthernetType: layers.EthernetTypeDot1Q},
		&layers.Dot1Q{VLANIdentifier: 3, Type: layers.EthernetTypeIPv6},
		&layers.IPv6{Version: 6, NextHeader: layers.IPProtocolUDP, SrcIP: ip62, DstIP: ip61},
		&layers.UDP{SrcPort: 2152, DstPort: 2152},
		makeGTPv1U(0x10000008, 1, 1),
		&layers.IPv4{Version: 4, TTL: 64, Protocol: layers.IPProtocolUDP, SrcIP: ip44, DstIP: ip43},
		&layers.UDP{SrcPort: 6363, DstPort: 6363},
	)
}

func TestLocatorTxHdr(t *testing.T) {
	assert, _ := makeAR(t)

	payload, _ := tlv.EncodeFrom(ndn.MakeInterest("/I"))
	checkTxHdr := func(locator string, expectedLayerTypes ...gopacket.LayerType) gopacket.Packet {
		loc := parseLocator(locator)
		txHdr := ethport.NewTxHdr(loc, false)
		pkt := makePacket(payload)
		defer pkt.Close()
		txHdr.Prepend(pkt, true)

		wire := pkt.Bytes()
		parsed := gopacket.NewPacket(wire, layers.LayerTypeEthernet, gopacket.NoCopy)
		expectedLayerTypes = append(expectedLayerTypes, ndnlayer.LayerTypeTLV, ndnlayer.LayerTypeNDN)
		ipLen, actualLayerTypes := 0, []gopacket.LayerType{}
		for i, l := range parsed.Layers() {
			if i < 2 {
				switch l.LayerType() {
				case layers.LayerTypeEthernet, layers.LayerTypeDot1Q:
					ipLen = len(l.LayerPayload()) - len(payload)
				}
			}
			actualLayerTypes = append(actualLayerTypes, l.LayerType())
		}
		assert.Equal(ipLen, txHdr.IPLen())
		assert.Equal(expectedLayerTypes, actualLayerTypes)

		var netLayer gopacket.NetworkLayer
		for _, l := range parsed.Layers() {
			switch l := l.(type) {
			case gopacket.NetworkLayer:
				netLayer = l
			case checksumTransportLayer:
				l.SetNetworkLayerForChecksum(netLayer) // ignore error when netLayer==nil
			}
		}
		e, mismatches := parsed.VerifyChecksums()
		assert.NoError(e)
		assert.Empty(mismatches)

		return parsed
	}

	checkTxHdr(`{
		"scheme": "ether",
		"local": "02:00:00:00:00:01",
		"remote": "02:00:00:00:00:02"
	}`, layers.LayerTypeEthernet)

	checkTxHdr(`{
		"scheme": "ether",
		"local": "02:00:00:00:00:01",
		"remote": "02:00:00:00:00:02",
		"vlan": 3
	}`, layers.LayerTypeEthernet, layers.LayerTypeDot1Q)

	udp4Pkt := checkTxHdr(`{
		"scheme": "udpe",
		"local": "02:00:00:00:00:01",
		"remote": "02:00:00:00:00:02",
		"localIP": "192.168.37.1",
		"remoteIP": "192.168.37.2",
		"localUDP": 6363,
		"remoteUDP": 16363
	}`, layers.LayerTypeEthernet, layers.LayerTypeIPv4, layers.LayerTypeUDP)
	udp4UDP := udp4Pkt.Layer(layers.LayerTypeUDP).(*layers.UDP)
	assert.Zero(udp4UDP.Checksum)

	udp6Pkt := checkTxHdr(`{
		"scheme": "udpe",
		"local": "02:00:00:00:00:01",
		"remote": "02:00:00:00:00:02",
		"localIP": "fde0:fd0a:3557:a8c7:db87:639f:9bd2:0001",
		"remoteIP": "fde0:fd0a:3557:a8c7:db87:639f:9bd2:0002",
		"localUDP": 6363,
		"remoteUDP": 16363
	}`, layers.LayerTypeEthernet, layers.LayerTypeIPv6, layers.LayerTypeUDP)
	udp6UDP := udp6Pkt.Layer(layers.LayerTypeUDP).(*layers.UDP)
	assert.NotZero(udp6UDP.Checksum)

	vxlanPkt := checkTxHdr(`{
		"scheme": "vxlan",
		"local": "02:00:00:00:00:01",
		"remote": "02:00:00:00:00:02",
		"localIP": "fde0:fd0a:3557:a8c7:db87:639f:9bd2:0001",
		"remoteIP": "fde0:fd0a:3557:a8c7:db87:639f:9bd2:0002",
		"vxlan": 0,
		"innerLocal": "02:00:00:00:00:03",
		"innerRemote": "02:00:00:00:00:04"
	}`, layers.LayerTypeEthernet, layers.LayerTypeIPv6, layers.LayerTypeUDP, layers.LayerTypeVXLAN, layers.LayerTypeEthernet)
	vxlanUDP := vxlanPkt.Layer(layers.LayerTypeUDP).(*layers.UDP)
	assert.GreaterOrEqual(uint16(vxlanUDP.SrcPort), uint16(0xC000))
	assert.EqualValues(4789, vxlanUDP.DstPort)

	gtp4Pkt := checkTxHdr(`{
		"scheme": "gtp",
		"local": "02:00:00:00:00:01",
		"remote": "02:00:00:00:00:02",
		"localIP": "192.168.37.1",
		"remoteIP": "192.168.37.2",
		"ulTEID": 268435464,
		"ulQFI": 1,
		"dlTEID": 536870920,
		"dlQFI": 11,
		"innerLocalIP": "192.168.60.3",
		"innerRemoteIP": "192.168.60.4"
	}`, layers.LayerTypeEthernet, layers.LayerTypeIPv4, layers.LayerTypeUDP, layers.LayerTypeGTPv1U, layers.LayerTypeIPv4, layers.LayerTypeUDP)
	gtp4OuterUDP := gtp4Pkt.Layer(layers.LayerTypeUDP).(*layers.UDP)
	assert.EqualValues(2152, gtp4OuterUDP.SrcPort)
	assert.EqualValues(2152, gtp4OuterUDP.DstPort)
	gtp4GTP := gtp4Pkt.Layer(layers.LayerTypeGTPv1U).(*layers.GTPv1U)
	assert.EqualValues(0x20000008, gtp4GTP.TEID)
	if assert.Len(gtp4GTP.GTPExtensionHeaders, 1) {
		gtp4PSC := gtp4GTP.GTPExtensionHeaders[0]
		assert.EqualValues(0x85, gtp4PSC.Type)
		assert.Equal([]byte{0x00, 0x0B}, gtp4PSC.Content)
	}

	gtp6Pkt := checkTxHdr(`{
		"scheme": "gtp",
		"local": "02:00:00:00:00:01",
		"remote": "02:00:00:00:00:02",
		"vlan": 3,
		"localIP": "fde0:fd0a:3557:a8c7:db87:639f:9bd2:0001",
		"remoteIP": "fde0:fd0a:3557:a8c7:db87:639f:9bd2:0002",
		"ulTEID": 268435464,
		"ulQFI": 1,
		"dlTEID": 536870920,
		"dlQFI": 11,
		"innerLocalIP": "192.168.60.3",
		"innerRemoteIP": "192.168.60.4"
	}`, layers.LayerTypeEthernet, layers.LayerTypeDot1Q, layers.LayerTypeIPv6, layers.LayerTypeUDP, layers.LayerTypeGTPv1U, layers.LayerTypeIPv4, layers.LayerTypeUDP)
	gtp6OuterUDP := gtp6Pkt.Layer(layers.LayerTypeUDP).(*layers.UDP)
	assert.NotZero(gtp6OuterUDP.Checksum)
	gtp6InnerUDP := gtp6Pkt.Layers()[6].(*layers.UDP)
	assert.EqualValues(an.UDPPortNDN, gtp6InnerUDP.SrcPort)
	assert.EqualValues(an.UDPPortNDN, gtp6InnerUDP.DstPort)
	assert.Zero(gtp6InnerUDP.Checksum)
}
