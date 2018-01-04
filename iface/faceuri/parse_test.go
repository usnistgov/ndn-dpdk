package faceuri

import "testing"

func TestParse(t *testing.T) {
	assert, _ := makeAR(t)

	tests := []struct {
		input  string
		ok     bool
		output string // "" indicates same as input
	}{
		{"dev://net_pcap1", true, ""},
		{"dev://net_pcap1/", true, "dev://net_pcap1"},
		{"dev://user@net_pcap1", false, ""},
		{"dev://net_pcap1:80", false, ""},
		{"dev://net_pcap1/path", false, ""},
		{"dev://net_pcap1?query", false, ""},
		{"dev://net_pcap1#fragment", false, ""},
		{"udp://192.0.2.1", true, "udp4://192.0.2.1:6363"},
		{"udp://192.0.2.1:7777", true, "udp4://192.0.2.1:7777"},
		{"udp4://192.0.2.1", true, "udp4://192.0.2.1:6363"},
		{"udp4://192.0.2.1:7777", true, ""},
		{"udp4://192.0.2.1/", true, "udp4://192.0.2.1:6363"},
		{"udp4://user@192.0.2.1", false, ""},
		{"udp4://0.0.0.0", false, ""},
		{"udp4://example.net", false, ""},
		{"udp4://255.255.255.255", false, ""},
		{"udp4://192.0.2.1:0", false, ""},
		{"udp4://192.0.2.1:77777", false, ""},
		{"udp4://192.0.2.1:dns", false, ""},
		{"udp4://192.0.2.1/path", false, ""},
		{"udp4://192.0.2.1?query", false, ""},
		{"udp4://192.0.2.1#fragment", false, ""},
		{"tcp://192.0.2.1", true, "tcp4://192.0.2.1:6363"},
		{"tcp://192.0.2.1:7777", true, "tcp4://192.0.2.1:7777"},
		{"tcp4://192.0.2.1", true, "tcp4://192.0.2.1:6363"},
		{"tcp4://192.0.2.1:7777", true, ""},
	}
	for _, tt := range tests {
		u, e := Parse(tt.input)
		if tt.ok {
			if assert.NoError(e, tt.input) && assert.NotNil(u, tt.input) {
				output := tt.output
				if output == "" {
					output = tt.input
				}
				assert.Equal(output, u.String())
			}
		} else {
			assert.Error(e, tt.input)
		}
	}
}
