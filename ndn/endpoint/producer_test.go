package endpoint_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"fmt"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/endpoint"
	"github.com/usnistgov/ndn-dpdk/ndn/keychain"
	"github.com/usnistgov/ndn-dpdk/ndn/keychain/eckey"
	"github.com/usnistgov/ndn-dpdk/ndn/l3"
)

func TestSignVerify(t *testing.T) {
	fw := l3.NewForwarder()
	assert, require := makeAR(t)

	makeSignerVerifier := func(name string) (ndn.Signer, ndn.Verifier) {
		keyName := keychain.ToKeyName(ndn.ParseName(name))
		pvt, e := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		require.NoError(e)
		signer, e := eckey.NewPrivateKey(keyName, pvt)
		require.NoError(e)
		verifier, e := eckey.NewPublicKey(keyName, &pvt.PublicKey)
		require.NoError(e)
		return signer, verifier
	}
	signer1, verifier1 := makeSignerVerifier("/K1")
	signer2, verifier2 := makeSignerVerifier("/K2")

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
	defer l3.DeleteDefaultForwarder()
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
	defer l3.DeleteDefaultForwarder()
	assert, require := makeAR(t)

	pCompleted, pCanceled := 0, 0
	pCtx, pCancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer pCancel()
	p, e := endpoint.Produce(pCtx, endpoint.ProducerOptions{
		Prefix: ndn.ParseName("/A"),
		Handler: func(ctx context.Context, interest ndn.Interest) (ndn.Data, error) {
			delay, _ := strconv.Atoi(string(interest.Name.Get(-1).Value))
			select {
			case <-time.After(time.Duration(delay) * time.Millisecond):
				pCompleted++
			case <-ctx.Done():
				pCanceled++
			}
			return ndn.MakeData(interest), nil
		},
	})
	require.NoError(e)
	defer p.Close()

	var cWait sync.WaitGroup
	cData, cExpire := 0, 0
	for i := 0; i < 250; i++ {
		cWait.Add(1)
		go func(i int) {
			defer cWait.Done()
			interest := ndn.MakeInterest(fmt.Sprintf("/A/%d", i), 300*time.Millisecond)
			data, e := endpoint.Consume(context.Background(), interest, endpoint.ConsumerOptions{})
			if data != nil {
				cData++
			} else if assert.EqualError(e, endpoint.ErrExpire.Error()) {
				cExpire++
			}
		}(i)
	}

	cWait.Wait()
	assert.Equal(250, cData+cExpire)
	assert.InDelta(250, pCompleted+pCanceled, 50)
	assert.InDelta(150, pCompleted, 50)
	assert.InDelta(pCompleted, cData, 50)
	assert.InDelta(pCanceled, cExpire, 50)
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
	defer l3.DeleteDefaultForwarder()
	assert, require := makeAR(t)

	var dest readvertiseDestinationMock
	l3.GetDefaultForwarder().AddReadvertiseDestination(&dest)

	p1, e := endpoint.Produce(context.Background(), endpoint.ProducerOptions{
		Prefix:  ndn.ParseName("/A"),
		Handler: producerHandlerNever,
	})
	require.NoError(e)
	if assert.Len(dest.advertised, 1) {
		nameEqual(assert, dest.advertised[0], "/A")
	}

	p2, e := endpoint.Produce(context.Background(), endpoint.ProducerOptions{
		Prefix:  ndn.ParseName("/A"),
		Handler: producerHandlerNever,
	})
	require.NoError(e)
	assert.Len(dest.advertised, 1)

	p1.Close()
	time.Sleep(50 * time.Millisecond)
	assert.Len(dest.withdrawn, 0)

	p2.Close()
	time.Sleep(50 * time.Millisecond) // producer.Close is asynchronous
	if assert.Len(dest.withdrawn, 1) {
		nameEqual(assert, dest.withdrawn[0], "/A")
	}
}

func TestProducerNoAdvertise(t *testing.T) {
	defer l3.DeleteDefaultForwarder()
	assert, require := makeAR(t)

	var dest readvertiseDestinationMock
	l3.GetDefaultForwarder().AddReadvertiseDestination(&dest)

	p, e := endpoint.Produce(context.Background(), endpoint.ProducerOptions{
		Prefix:      ndn.ParseName("/A"),
		NoAdvertise: true,
		Handler:     producerHandlerNever,
	})
	require.NoError(e)
	assert.Len(dest.advertised, 0)

	p.Close()
	time.Sleep(50 * time.Millisecond) // producer.Close is asynchronous
	assert.Len(dest.withdrawn, 0)
}
