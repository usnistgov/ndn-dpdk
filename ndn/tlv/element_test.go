package tlv_test

import (
	"testing"

	"github.com/usnistgov/ndn-dpdk/ndn/ndntestvector"
	"github.com/usnistgov/ndn-dpdk/ndn/tlv"
)

func TestElement(t *testing.T) {
	assert, _ := makeAR(t)

	for _, tt := range ndntestvector.TlvElementTests {
		input := bytesFromHex(tt.Input)
		var element tlv.Element
		rest, e := element.Decode(input)

		if tt.Bad {
			assert.Error(e, tt.Input)
		} else if assert.NoError(e, tt.Input) {
			assert.Equal(len(input), element.Size(), tt.Input)
			assert.Len(rest, 0, tt.Input)

			assert.Equal(tt.Type, element.Type, tt.Input)
			value := bytesFromHex(tt.Value)
			assert.Equal(len(value), element.Length(), tt.Input)
			bytesEqual(assert, value, element.Value, tt.Input)

			var nni tlv.NNI
			if e := nni.UnmarshalBinary(value); e == nil {
				assert.EqualValues(tt.Nni, nni, tt.Input)
				assert.Equal(len(value), nni.Size())
				nniV := nni.Encode(nil)
				assert.Equal(value, nniV)
			} else {
				assert.True(tt.Nni == ndntestvector.NotNni, tt.Input)
			}
		}
	}
}
