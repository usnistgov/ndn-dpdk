package endpoint_test

import (
	"context"
	"testing"
	"time"

	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/endpoint"
	"github.com/usnistgov/ndn-dpdk/ndn/l3"
	"go4.org/must"
)

func addRetxLimitTestProducer(invokeCount *int) (endpoint.Producer, error) {
	return endpoint.Produce(context.Background(), endpoint.ProducerOptions{
		Prefix: ndn.ParseName("/A"),
		Handler: func(ctx context.Context, interest ndn.Interest) (ndn.Data, error) {
			if *invokeCount++; *invokeCount <= 2 {
				return ndn.Data{}, nil
			}
			return ndn.MakeData(interest.Name), nil
		},
	})
}

func TestRetxLimit(t *testing.T) {
	t.Cleanup(l3.DeleteDefaultForwarder)
	assert, require := makeAR(t)

	tests := []struct {
		retx       endpoint.RetxPolicy
		nInterests int
	}{
		{nil, 1},
		{endpoint.RetxOptions{}, 1},
		{endpoint.RetxOptions{Limit: 1, Interval: 50 * time.Millisecond}, 2},  // retx before timeout
		{endpoint.RetxOptions{Limit: 1, Interval: 400 * time.Millisecond}, 2}, // retx after timeout
	}
	for i, tt := range tests {
		var invokeCount int
		p, e := addRetxLimitTestProducer(&invokeCount)
		require.NoError(e, "%d", i)

		data, e := endpoint.Consume(context.Background(), ndn.MakeInterest("/A", 200*time.Millisecond),
			endpoint.ConsumerOptions{Retx: tt.retx})
		assert.Nil(data, "%d", i)
		assert.EqualError(e, endpoint.ErrExpire.Error(), "%d", i)

		assert.Equal(tt.nInterests, invokeCount)
		must.Close(p)
		l3.DeleteDefaultForwarder()
	}
}

func TestConsumerCancel(t *testing.T) {
	t.Cleanup(l3.DeleteDefaultForwarder)
	assert, require := makeAR(t)

	var invokeCount int
	p, e := addRetxLimitTestProducer(&invokeCount)
	require.NoError(e)
	defer p.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	data, e := endpoint.Consume(ctx, ndn.MakeInterest("/A", 200*time.Millisecond),
		endpoint.ConsumerOptions{Retx: endpoint.RetxOptions{Limit: 2}})
	assert.Nil(data)
	assert.EqualError(e, context.DeadlineExceeded.Error())
}
