package fileserver_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"math"
	"os"
	"sync"
	"testing"
	"time"

	mathpkg "github.com/pkg/math"
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
	"golang.org/x/sys/unix"
)

func TestServer(t *testing.T) {
	assert, require := makeAR(t)

	face := intface.MustNew()
	defer face.D.Close()

	segmentLen := 3000
	cfg := fileserver.Config{
		NThreads: 2,
		Mounts: []fileserver.Mount{
			{Prefix: ndn.ParseName("/usr/bin"), Path: "/usr/bin"},
			{Prefix: ndn.ParseName("/usr/local-bin"), Path: "/usr/local/bin"},
			{Prefix: ndn.ParseName("/usr/local-lib"), Path: "/usr/local/lib"},
		},
		SegmentLen:   segmentLen,
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
	fetchMetadata := func(name string) (m fileserver.Metadata, e error) {
		mName := ndn.ParseName(name)
		mName = append(mName, fileserver.KeywordMetadata)
		mInterest := ndn.MakeInterest(mName, ndn.CanBePrefixFlag, ndn.MustBeFreshFlag)
		mData, e := endpoint.Consume(timeout, mInterest, endpoint.ConsumerOptions{
			Fw:   fw,
			Retx: endpoint.RetxOptions{Limit: 3},
		})
		if e != nil {
			return m, e
		}
		e = m.UnmarshalBinary(mData.Content)
		return m, e
	}
	fetchPayload := func(m fileserver.Metadata, lastSeg *tlv.NNI) (payload []byte, e error) {
		fOpts := segmented.FetchOptions{
			Fw:        fw,
			RetxLimit: 3,
			MaxCwnd:   256,
		}
		if lastSeg != nil {
			fOpts.SegmentEnd = 1 + uint64(*lastSeg)
		}
		fetcher := segmented.Fetch(m.Versioned, fOpts)
		return fetcher.Payload(timeout)
	}
	testFetchFile := func(filename, name string, setSegmentEnd bool) {
		defer wg.Done()
		content, e := os.ReadFile(filename)
		require.NoError(e)
		digest := sha256.Sum256(content)

		m, e := fetchMetadata(name)
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

		fetcherLastSeg := &lastSeg
		if !setSegmentEnd {
			fetcherLastSeg = nil
		}
		payload, e := fetchPayload(m, fetcherLastSeg)
		assert.NoError(e)
		assert.Len(payload, len(content))
		assert.Equal(digest, sha256.Sum256(payload))
	}
	testFetchDir := func(dirname, name string) {
		defer wg.Done()
		dirEntries, e := os.ReadDir(dirname)
		require.NoError(e)
		dirEntryNames := map[string]bool{}
		for _, dirEntry := range dirEntries {
			filename, mode := dirEntry.Name(), dirEntry.Type()
			switch {
			case mode.IsRegular():
				dirEntryNames[filename] = false
			case mode.IsDir():
				dirEntryNames[filename] = true
			}
		}

		m, e := fetchMetadata(name)
		if !assert.NoError(e) {
			return
		}
		assert.False(m.FinalBlock.Valid())
		assert.False(m.Mtime.IsZero())

		payload, e := fetchPayload(m, nil)
		assert.NoError(e)

		nFound := 0
		for _, line := range bytes.Split(bytes.TrimSuffix(payload, []byte{0}), []byte{0}) {
			filename := bytes.TrimSuffix(line, []byte{'/'})
			isDir, ok := dirEntryNames[string(filename)]
			if assert.True(ok, "%s", string(filename)) {
				assert.Equal(isDir, bytes.HasSuffix(line, []byte{'/'}), "%s", string(filename))
			}
			nFound++
		}
		assert.GreaterOrEqual(nFound, mathpkg.MinInt(segmentLen/(unix.NAME_MAX+2), len(dirEntryNames)))
	}

	wg.Add(4)
	go testFetchFile("/usr/local/bin/dpdk-testpmd", "/usr/local-bin/dpdk-testpmd", true)
	go testFetchFile("/usr/bin/jq", "/usr/bin/jq", false)
	go testFetchDir("/usr/bin", "/usr/bin")
	go testFetchDir("/usr/local/bin", "/usr/local-bin/"+fileserver.KeywordLs.String())
	wg.Wait()
}
