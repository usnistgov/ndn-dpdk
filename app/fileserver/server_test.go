package fileserver_test

import (
	"context"
	"crypto/sha256"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/usnistgov/ndn-dpdk/app/fileserver"
	"github.com/usnistgov/ndn-dpdk/app/tgtestenv"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealthread"
	"github.com/usnistgov/ndn-dpdk/iface/intface"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/l3"
	"github.com/usnistgov/ndn-dpdk/ndn/segmented"
)

func TestServer(t *testing.T) {
	defer ealthread.AllocClear()
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
		SegmentLen: 3000,
	}

	p, e := fileserver.New(face.D, cfg)
	require.NoError(e)
	defer p.Close()
	require.NoError(ealthread.AllocThread(p.Workers()...))
	p.ConnectRxQueues(tgtestenv.DemuxI)
	p.Launch()
	time.Sleep(time.Second)

	fw := l3.NewForwarder()
	fwFace, e := fw.AddFace(face.A)
	require.NoError(e)
	fwFace.AddRoute(ndn.ParseName("/F"))

	var wg sync.WaitGroup
	timeout, cancel := context.WithTimeout(context.TODO(), 20*time.Second)
	defer cancel()
	testFetchFile := func(filename, name string) {
		defer wg.Done()
		content, e := os.ReadFile(filename)
		require.NoError(e)
		digest := sha256.Sum256(content)

		fetcher := segmented.Fetch(ndn.ParseName(name), segmented.FetchOptions{
			Fw:        fw,
			RetxLimit: 3,
			MaxCwnd:   256,
		})
		payload, e := fetcher.Payload(timeout)
		assert.NoError(e)
		assert.Len(payload, len(content))
		assert.Equal(digest, sha256.Sum256(payload))
	}

	wg.Add(2)
	go testFetchFile("/usr/local/bin/dpdk-testpmd", "/usr/local-bin/dpdk-testpmd")
	go testFetchFile("/usr/bin/jq", "/usr/bin/jq")
	wg.Wait()
}
