package endpoint_test

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/endpoint"
	"github.com/usnistgov/ndn-dpdk/ndn/keychain"
	"github.com/usnistgov/ndn-dpdk/ndn/l3"
	"github.com/usnistgov/ndn-dpdk/ndn/ndntestenv"
	"github.com/usnistgov/ndn-dpdk/ndn/tlv"
	"go4.org/must"
)

func TestSignVerify(t *testing.T) {
	fw := l3.NewForwarder()
	assert, require := makeAR(t)

	signer1, verifier1, e := keychain.NewECDSAKeyPair(ndn.ParseName("/K1"))
	require.NoError(e)
	signer2, verifier2, e := keychain.NewECDSAKeyPair(ndn.ParseName("/K2"))
	require.NoError(e)

	p, e := endpoint.Produce(context.Background(), endpoint.ProducerOptions{
		Prefix: ndn.ParseName("/A"),
		Handler: func(ctx context.Context, interest ndn.Interest) (ndn.Data, error) {
			data := ndn.MakeData(interest.Name)
			if interest.Name.Get(-1).Value[0] == '2' {
				e := signer2.Sign(&data)
				require.NoError(e)
			}
			return data, nil
		},
		Fw:         fw,
		DataSigner: signer1,
	})
	require.NoError(e)
	defer p.Close()

	data1, e := endpoint.Consume(context.Background(), ndn.MakeInterest("/A/1"),
		endpoint.ConsumerOptions{Fw: fw, Verifier: verifier1})
	if assert.NoError(e) {
		nameEqual(assert, "/A/1", data1)
	}

	_, e = endpoint.Consume(context.Background(), ndn.MakeInterest("/A/1"),
		endpoint.ConsumerOptions{Fw: fw, Verifier: verifier2})
	if assert.Error(e) {
		assert.NotEqual(endpoint.ErrExpire.Error(), e.Error())
	}

	data2, e := endpoint.Consume(context.Background(), ndn.MakeInterest("/A/2"),
		endpoint.ConsumerOptions{Fw: fw, Verifier: verifier2})
	if assert.NoError(e) {
		nameEqual(assert, "/A/2", data2)
	}
}

func TestProducerNonMatch(t *testing.T) {
	t.Cleanup(l3.DeleteDefaultForwarder)
	assert, require := makeAR(t)

	p, e := endpoint.Produce(context.Background(), endpoint.ProducerOptions{
		Prefix: ndn.ParseName("/A"),
		Handler: func(ctx context.Context, interest ndn.Interest) (ndn.Data, error) {
			return ndn.MakeData("/A/0"), nil
		},
	})
	require.NoError(e)
	defer p.Close()

	data, e := endpoint.Consume(context.Background(), ndn.MakeInterest("/A/9", 100*time.Millisecond),
		endpoint.ConsumerOptions{})
	assert.Nil(data)
	assert.EqualError(e, endpoint.ErrExpire.Error())
}

func TestProducerConcurrent(t *testing.T) {
	t.Cleanup(l3.DeleteDefaultForwarder)
	assert, require := makeAR(t)

	var pCompleted, pCanceled atomic.Int32
	pCtx, pCancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer pCancel()
	p, e := endpoint.Produce(pCtx, endpoint.ProducerOptions{
		Prefix: ndn.ParseName("/A"),
		Handler: func(ctx context.Context, interest ndn.Interest) (ndn.Data, error) {
			delay, _ := strconv.Atoi(string(interest.Name.Get(-1).Value))
			select {
			case <-time.After(time.Duration(delay) * time.Millisecond):
				pCompleted.Add(1)
			case <-ctx.Done():
				pCanceled.Add(1)
			}
			return ndn.MakeData(interest), nil
		},
	})
	require.NoError(e)
	defer p.Close()

	var cWait sync.WaitGroup
	var cData, cExpire atomic.Int32
	for i := range 250 {
		cWait.Add(1)
		go func(i int) {
			defer cWait.Done()
			interest := ndn.MakeInterest(fmt.Sprintf("/A/%d", i), 300*time.Millisecond)
			data, e := endpoint.Consume(context.Background(), interest, endpoint.ConsumerOptions{})
			if data != nil {
				cData.Add(1)
			} else if assert.EqualError(e, endpoint.ErrExpire.Error()) {
				cExpire.Add(1)
			}
		}(i)
	}

	cWait.Wait()
	assert.EqualValues(250, cData.Load()+cExpire.Load())
	assert.InDelta(250, pCompleted.Load()+pCanceled.Load(), 70)
	assert.InDelta(150, pCompleted.Load(), 70)
	assert.InDelta(pCompleted.Load(), cData.Load(), 70)
	assert.InDelta(pCanceled.Load(), cExpire.Load(), 70)
}

func TestProducerPreEncoded(t *testing.T) {
	assert, require := makeAR(t)
	bridge := ndntestenv.NewBridge(ndntestenv.BridgeConfig{})

	pWire := bytesFromHex(`
		0615
			0706080141080130
			D4014D
			1606
				1B01C8
				D2012D
			1700
	`)
	var pPkt ndn.Packet
	e := tlv.Decode(pWire, &pPkt)
	require.NoError(e)
	require.NotNil(pPkt.Data)
	pData := *pPkt.Data

	p, e := endpoint.Produce(context.Background(), endpoint.ProducerOptions{
		Prefix: ndn.ParseName("/A"),
		Handler: func(ctx context.Context, interest ndn.Interest) (ndn.Data, error) {
			return pData, nil
		},
		Fw: bridge.FwA,
	})
	require.NoError(e)
	defer p.Close()

	cData, e := endpoint.Consume(context.Background(), ndn.MakeInterest("/A/0"), endpoint.ConsumerOptions{
		Fw: bridge.FwB,
	})
	require.NoError(e)

	cWire, e := cData.ToPacket().Field().AppendBinary(nil)
	require.NoError(e)
	bytesEqual(assert, pWire, cWire)
}

var producerHandlerNever endpoint.ProducerHandler = func(ctx context.Context, interest ndn.Interest) (ndn.Data, error) {
	panic("this ProducerHandler should not be invoked")
}

type readvertiseDestinationMock struct {
	advertised []ndn.Name
	withdrawn  []ndn.Name
}

func (dest *readvertiseDestinationMock) Advertise(prefix ndn.Name) error {
	dest.advertised = append(dest.advertised, prefix)
	return nil
}

func (dest *readvertiseDestinationMock) Withdraw(prefix ndn.Name) error {
	dest.withdrawn = append(dest.withdrawn, prefix)
	return nil
}

func TestProducerAdvertise(t *testing.T) {
	t.Cleanup(l3.DeleteDefaultForwarder)
	assert, require := makeAR(t)

	var dest readvertiseDestinationMock
	l3.GetDefaultForwarder().AddReadvertiseDestination(&dest)

	p1, e := endpoint.Produce(context.Background(), endpoint.ProducerOptions{
		Prefix:  ndn.ParseName("/A"),
		Handler: producerHandlerNever,
	})
	require.NoError(e)
	time.Sleep(50 * time.Millisecond)
	if assert.Len(dest.advertised, 1) {
		nameEqual(assert, dest.advertised[0], "/A")
	}

	p2, e := endpoint.Produce(context.Background(), endpoint.ProducerOptions{
		Prefix:  ndn.ParseName("/A"),
		Handler: producerHandlerNever,
	})
	require.NoError(e)
	time.Sleep(50 * time.Millisecond)
	assert.Len(dest.advertised, 1)

	must.Close(p1)
	time.Sleep(50 * time.Millisecond)
	assert.Len(dest.withdrawn, 0)

	must.Close(p2)
	time.Sleep(50 * time.Millisecond)
	if assert.Len(dest.withdrawn, 1) {
		nameEqual(assert, dest.withdrawn[0], "/A")
	}
}

func TestProducerNoAdvertise(t *testing.T) {
	t.Cleanup(l3.DeleteDefaultForwarder)
	assert, require := makeAR(t)

	var dest readvertiseDestinationMock
	l3.GetDefaultForwarder().AddReadvertiseDestination(&dest)

	p, e := endpoint.Produce(context.Background(), endpoint.ProducerOptions{
		Prefix:      ndn.ParseName("/A"),
		NoAdvertise: true,
		Handler:     producerHandlerNever,
	})
	require.NoError(e)
	time.Sleep(50 * time.Millisecond)
	assert.Len(dest.advertised, 0)

	must.Close(p)
	time.Sleep(50 * time.Millisecond)
	assert.Len(dest.withdrawn, 0)
}
