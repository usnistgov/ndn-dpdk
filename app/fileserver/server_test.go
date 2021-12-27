package fileserver_test

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
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
	"github.com/usnistgov/ndn-dpdk/ndn/rdr"
	"github.com/usnistgov/ndn-dpdk/ndn/rdr/ndn6file"
	"github.com/usnistgov/ndn-dpdk/ndn/segmented"
	"github.com/usnistgov/ndn-dpdk/ndn/tlv"
	"golang.org/x/sys/unix"
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
		KeepFds:      4,
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
	fwFace.AddRoute(ndn.ParseName("/"))

	var wg sync.WaitGroup
	timeout, cancel := context.WithTimeout(context.TODO(), 20*time.Second)
	defer cancel()
	metadataOpts := endpoint.ConsumerOptions{
		Fw:   fw,
		Retx: endpoint.RetxOptions{Limit: 3},
	}
	fetchPayload := func(m ndn6file.Metadata, lastSeg *tlv.NNI) (payload []byte, e error) {
		fOpts := segmented.FetchOptions{
			Fw:        fw,
			RetxLimit: 3,
			MaxCwnd:   256,
		}
		if lastSeg != nil {
			fOpts.SegmentEnd = 1 + uint64(*lastSeg)
		}
		fetcher := segmented.Fetch(m.Name, fOpts)
		return fetcher.Payload(timeout)
	}
	testFetchFile := func(filename, name string, setSegmentEnd bool) {
		defer wg.Done()
		content, e := os.ReadFile(filename)
		require.NoError(e)
		digest := sha256.Sum256(content)

		var m ndn6file.Metadata
		if !assert.NoError(rdr.RetrieveMetadata(timeout, &m, ndn.ParseName(name), metadataOpts)) {
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
		if !assert.NoError(e) {
			return
		}
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

		var m ndn6file.Metadata
		if !assert.NoError(rdr.RetrieveMetadata(timeout, &m, ndn.ParseName(name), metadataOpts)) {
			return
		}
		assert.False(m.FinalBlock.Valid())
		assert.False(m.Mtime.IsZero())

		payload, e := fetchPayload(m, nil)
		assert.NoError(e)
		var ls ndn6file.DirectoryListing
		if e := ls.UnmarshalBinary(payload); !assert.NoError(e) {
			return
		}

		nFound := 0
		for _, entry := range ls {
			filename := entry.Name()
			isDir, ok := dirEntryNames[filename]
			if assert.True(ok, "%s", string(filename)) {
				assert.Equal(isDir, entry.IsDir(), "%s", string(filename))
			}
			nFound++
		}
		assert.GreaterOrEqual(nFound, mathpkg.MinInt(cfg.SegmentLen/(unix.NAME_MAX+2), len(dirEntryNames)))
	}
	testNotFound := func(name string, expectNack bool) {
		defer wg.Done()
		expectedErr := endpoint.ErrExpire
		if expectNack {
			expectedErr = ndn.ErrContentType
		}

		var m ndn6file.Metadata
		e := rdr.RetrieveMetadata(timeout, &m, ndn.ParseName(name), metadataOpts)
		assert.ErrorIs(e, expectedErr, "%s", name)
	}

	assert.NoFileExists("/usr/local/bin/ndndpdk/no-such-program")
	localLibs := func() (list []string) {
		dirEntries, e := os.ReadDir("/usr/local/lib")
		require.NoError(e)
		for _, dirEntry := range dirEntries {
			if dirEntry.Type().IsRegular() {
				list = append(list, dirEntry.Name())
			}
		}
		require.GreaterOrEqual(len(dirEntries), cfg.NThreads*cfg.KeepFds)
		return
	}()

	wg.Add(12 + len(localLibs))
	go testFetchFile("/usr/local/bin/dpdk-testpmd", "/usr/local-bin/dpdk-testpmd", true)
	go testFetchFile("/usr/bin/jq", "/usr/bin/jq", false)
	go testFetchDir("/usr/bin", "/usr/bin")
	go testFetchDir("/usr/local/bin", "/usr/local-bin/"+ndn6file.KeywordLs.String())
	go testNotFound("/usr/local-bin/ndndpdk/no-such-program", true)
	go testNotFound("/no-such-mount/autoexec.bat", false)
	go testNotFound("/usr/local-bin/bad/zero%00/filename", false)
	go testNotFound("/usr/local-bin/bad/slash%2F/filename", false)
	go testNotFound("/usr/local-bin/bad/.../filename", false)
	go testNotFound("/usr/local-bin/bad/..../filename", false)
	go testNotFound("/usr/local-bin/bad/...../filename", false)
	go testNotFound("/usr/local-bin/bad/....../filename", true)
	for _, localLib := range localLibs {
		// cannot get file metadata with 32=ls component, but this opens file descriptor for testing keepFds limit
		go testNotFound("/usr/local-lib/"+localLib+"/"+ndn6file.KeywordLs.String(), true)
	}
	wg.Wait()

	cnt := p.Counters()
	assert.Greater(cnt.ReqRead, uint64(0))
	assert.Greater(cnt.ReqLs, uint64(0))
	assert.Greater(cnt.ReqMetadata, uint64(0))
	assert.Greater(cnt.FdNew, uint64(0))
	assert.Greater(cnt.FdNotFound, uint64(0))
	assert.Greater(cnt.FdClose, uint64(0))
	assert.LessOrEqual(cnt.FdNew, cnt.FdClose+uint64(cfg.NThreads*cfg.KeepFds))
	assert.Greater(cnt.UringSubmit, uint64(0))
	assert.Greater(cnt.UringSubmitNonBlock, uint64(0))
	assert.Greater(cnt.SqeSubmit, uint64(0))
	cntJ, _ := json.Marshal(cnt)
	fmt.Println(string(cntJ))
}
