package ndntestvector

// NotNni in TlvElementTests indicates the TLV-VALUE is not a non-negative integer.
const NotNni uint64 = 0xB9C0CEA091E491F0

// TlvElementTests contains test vectors for TLV element decoder.
var TlvElementTests = []struct {
	Input string
	Bad   bool
	Type  uint32
	Value string
	Nni   uint64
}{
	{Input: "", Bad: true},                               // empty
	{Input: "01", Bad: true},                             // missing TLV-LENGTH
	{Input: "01 01", Bad: true},                          // incomplete TLV-VALUE
	{Input: "01 FD00", Bad: true},                        // incomplete TLV-LENGTH
	{Input: "01 FF0000000100000000 A0", Bad: true},       // TLV-LENGTH overflow
	{Input: "01 04 A0A1", Bad: true},                     // incomplete TLV-VALUE
	{Input: "00 00", Bad: true},                          // zero TLV-TYPE
	{Input: "FF0000000100000000 00", Bad: true},          // too large TLV-TYPE
	{Input: "01 00", Type: 0x01, Value: "", Nni: NotNni}, // zero TLV-LENGTH
	{Input: "FC 01 01", Type: 0xFC, Value: "01", Nni: 0x01},
	{Input: "FD00FD 02 A0A1", Type: 0xFD, Value: "A0A1", Nni: 0xA0A1},
	{Input: "FD00FF 03 A0A1A2", Type: 0xFF, Value: "A0A1A2", Nni: NotNni},
	{Input: "FE00010000 04 A0A1A2A3", Type: 0x10000, Value: "A0A1A2A3", Nni: 0xA0A1A2A3},
	{Input: "FEFFFFFFFF 05 A0A1A2A3A4", Type: 0xFFFFFFFF, Value: "A0A1A2A3A4", Nni: NotNni},
	{Input: "01 06 A0A1A2A3A4A5", Type: 0x01, Value: "A0A1A2A3A4A5", Nni: NotNni},
	{Input: "01 07 A0A1A2A3A4A5A6", Type: 0x01, Value: "A0A1A2A3A4A5A6", Nni: NotNni},
	{Input: "01 08 A0A1A2A3A4A5A6A7", Type: 0x01, Value: "A0A1A2A3A4A5A6A7", Nni: 0xA0A1A2A3A4A5A6A7},
	{Input: "01 09 A0A1A2A3A4A5A6A7A8", Type: 0x01, Value: "A0A1A2A3A4A5A6A7A8", Nni: NotNni},
}
