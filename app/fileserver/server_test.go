package fileserver_test

import (
	"context"
	"crypto/sha256"
	"math"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/usnistgov/ndn-dpdk/app/fileserver"
	"github.com/usnistgov/ndn-dpdk/app/tg/tgtestenv"
	"github.com/usnistgov/ndn-dpdk/core/nnduration"
	"github.com/usnistgov/ndn-dpdk/iface/intface"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/an"
	"github.com/usnistgov/ndn-dpdk/ndn/endpoint"
	"github.com/usnistgov/ndn-dpdk/ndn/l3"
	"github.com/usnistgov/ndn-dpdk/ndn/segmented"
	"github.com/usnistgov/ndn-dpdk/ndn/tlv"
)

func TestServer(t *testing.T) {
	assert, require := makeAR(t)

	face := intface.MustNew()
	defer face.D.Close()

	cfg := fileserver.Config{
		NThreads: 2,
		Mounts: []fileserver.Mount{
			{Prefix: ndn.ParseName("/usr/bin"), Path: "/usr/bin"},
			{Prefix: ndn.ParseName("/usr/local-bin"), Path: "/usr/local/bin"},
			{Prefix: ndn.ParseName("/usr/local-lib"), Path: "/usr/local/lib"},
		},
		SegmentLen:   3000,
		StatValidity: nnduration.Nanoseconds(100 * time.Millisecond),
	}

	p, e := fileserver.New(face.D, cfg)
	require.NoError(e)
	defer p.Close()
	tgtestenv.Open(t, p)
	p.Launch()
	time.Sleep(time.Second)

	fw := l3.NewForwarder()
	fwFace, e := fw.AddFace(face.A)
	require.NoError(e)
	fwFace.AddRoute(ndn.ParseName("/F"))

	var wg sync.WaitGroup
	timeout, cancel := context.WithTimeout(context.TODO(), 20*time.Second)
	defer cancel()
	testFetchFile := func(filename, name string, setSegmentEnd bool) {
		defer wg.Done()
		content, e := os.ReadFile(filename)
		require.NoError(e)
		digest := sha256.Sum256(content)

		mName := ndn.ParseName(name)
		mName = append(mName, fileserver.KeywordMetadata)
		mInterest := ndn.MakeInterest(mName, ndn.CanBePrefixFlag, ndn.MustBeFreshFlag)
		mData, e := endpoint.Consume(timeout, mInterest, endpoint.ConsumerOptions{
			Fw:   fw,
			Retx: endpoint.RetxOptions{Limit: 3},
		})
		if !assert.NoError(e) {
			return
		}
		var m fileserver.Metadata
		e = m.UnmarshalBinary(mData.Content)
		if !assert.NoError(e) {
			return
		}
		lastSeg := tlv.NNI(math.MaxUint64)
		if assert.True(m.FinalBlock.Valid()) {
			assert.EqualValues(an.TtSegmentNameComponent, m.FinalBlock.Type)
			assert.NoError(lastSeg.UnmarshalBinary(m.FinalBlock.Value))
		}
		assert.EqualValues(cfg.SegmentLen, m.SegmentSize)
		assert.EqualValues(len(content), m.Size)
		assert.False(m.Mtime.IsZero())

		fOpts := segmented.FetchOptions{
			Fw:        fw,
			RetxLimit: 3,
			MaxCwnd:   256,
		}
		if setSegmentEnd {
			fOpts.SegmentEnd = 1 + uint64(lastSeg)
		}
		fetcher := segmented.Fetch(m.Versioned, fOpts)
		payload, e := fetcher.Payload(timeout)
		assert.NoError(e)
		assert.Len(payload, len(content))
		assert.Equal(digest, sha256.Sum256(payload))
	}

	wg.Add(2)
	go testFetchFile("/usr/local/bin/dpdk-testpmd", "/usr/local-bin/dpdk-testpmd", true)
	go testFetchFile("/usr/bin/jq", "/usr/bin/jq", false)
	wg.Wait()
}
