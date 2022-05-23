package fileserver_test

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"math"
	"os"
	"path"
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
	"github.com/usnistgov/ndn-dpdk/ndn/rdr"
	"github.com/usnistgov/ndn-dpdk/ndn/rdr/ndn6file"
	"github.com/usnistgov/ndn-dpdk/ndn/segmented"
	"github.com/usnistgov/ndn-dpdk/ndn/tlv"
)

func TestServer(t *testing.T) {
	assert, require := makeAR(t)

	face := intface.MustNew()
	t.Cleanup(func() { face.D.Close() })

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
	t.Cleanup(func() { p.Close() })
	tgtestenv.Open(t, p)
	p.Launch()
	time.Sleep(time.Second)

	fw := l3.NewForwarder()
	fwFace, e := fw.AddFace(face.A)
	require.NoError(e)
	fwFace.AddRoute(ndn.ParseName("/"))

	timeout, cancel := context.WithTimeout(context.TODO(), 20*time.Second)
	t.Cleanup(cancel)
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

	for _, tt := range []struct {
		Filename      string
		Name          string
		SetSegmentEnd bool
	}{
		{"/usr/local/bin/dpdk-testpmd", "/usr/local-bin/dpdk-testpmd", true},
		{"/usr/bin/jq", "/usr/bin/jq", false},
	} {
		tt := tt
		t.Run(tt.Filename, func(t *testing.T) {
			t.Parallel()
			assert, require := makeAR(t)

			content, e := os.ReadFile(tt.Filename)
			require.NoError(e)
			digest := sha256.Sum256(content)

			var m ndn6file.Metadata
			if !assert.NoError(rdr.RetrieveMetadata(timeout, &m, ndn.ParseName(tt.Name), metadataOpts)) {
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
			if !tt.SetSegmentEnd {
				fetcherLastSeg = nil
			}
			payload, e := fetchPayload(m, fetcherLastSeg)
			if !assert.NoError(e) {
				return
			}
			assert.Len(payload, len(content))
			assert.Equal(digest, sha256.Sum256(payload))
		})
	}

	for _, tt := range []struct {
		Dirname string
		Name    string
	}{
		{"/usr/bin", "/usr/bin"},
		{"/usr/local/bin", "/usr/local-bin/" + ndn6file.KeywordLs.String()},
	} {
		tt := tt
		t.Run(tt.Dirname, func(t *testing.T) {
			t.Parallel()
			assert, require := makeAR(t)

			dirEntries, e := os.ReadDir(tt.Dirname)
			require.NoError(e)
			dirEntryNames := map[string]bool{}
			for _, dirEntry := range dirEntries {
				filename, mode := dirEntry.Name(), dirEntry.Type()
				if mode&os.ModeSymlink != 0 {
					if info, e := os.Stat(path.Join(tt.Dirname, filename)); e == nil {
						mode = info.Mode()
					}
				}
				switch {
				case mode.IsRegular():
					dirEntryNames[filename] = false
				case mode.IsDir():
					dirEntryNames[filename] = true
				}
			}

			var m ndn6file.Metadata
			if !assert.NoError(rdr.RetrieveMetadata(timeout, &m, ndn.ParseName(tt.Name), metadataOpts)) {
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

			for _, entry := range ls {
				filename := entry.Name()
				isDir, ok := dirEntryNames[filename]
				assert.True(ok, filename)
				assert.Equal(isDir, entry.IsDir(), filename)
				delete(dirEntryNames, filename)
			}
			assert.Empty(dirEntryNames)
		})
	}

	assert.NoFileExists("/usr/local/bin/ndndpdk-no-such-program")
	type notFoundTest struct {
		Name       string
		ExpectNack bool
	}
	notFoundTests := []notFoundTest{
		{"/usr/local-bin/ndndpdk-no-such-program", true},
		{"/no-such-mount/autoexec.bat", false},
		{"/usr/local-bin/bad/zero%00/filename", false},
		{"/usr/local-bin/bad/slash%2F/filename", false},
		{"/usr/local-bin/bad/.../filename", false},
		{"/usr/local-bin/bad/..../filename", false},
		{"/usr/local-bin/bad/...../filename", false},
		{"/usr/local-bin/bad/....../filename", true},
	}
	if dirEntries, e := os.ReadDir("/usr/local/lib"); assert.NoError(e) {
		for _, dirEntry := range dirEntries {
			if dirEntry.Type().IsRegular() {
				// cannot get file metadata with 32=ls component, but this opens file descriptor for testing keepFds limit
				notFoundTests = append(notFoundTests, notFoundTest{
					Name:       "/usr/local-lib/" + dirEntry.Name() + "/" + ndn6file.KeywordLs.String(),
					ExpectNack: true,
				})
			}
		}
		require.GreaterOrEqual(len(notFoundTests), cfg.NThreads*cfg.KeepFds)
	}

	for _, tt := range notFoundTests {
		tt := tt
		t.Run(tt.Name, func(t *testing.T) {
			t.Parallel()
			assert, _ := makeAR(t)

			expectedErr := endpoint.ErrExpire
			if tt.ExpectNack {
				expectedErr = ndn.ErrContentType
			}

			var m ndn6file.Metadata
			e := rdr.RetrieveMetadata(timeout, &m, ndn.ParseName(tt.Name), metadataOpts)
			assert.ErrorIs(e, expectedErr)
		})
	}

	t.Cleanup(func() {
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
		t.Log(string(cntJ))
	})
}
