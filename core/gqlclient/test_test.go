package gqlclient_test

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/gabstv/freeport"
	"github.com/usnistgov/ndn-dpdk/core/gqlserver"
	"github.com/usnistgov/ndn-dpdk/core/testenv"
)

var (
	makeAR = testenv.MakeAR

	serverURI string
)

func TestMain(m *testing.M) {
	port, e := freeport.TCP()
	if e != nil {
		panic(e)
	}
	os.Setenv("GQLSERVER_HTTP", fmt.Sprintf("127.0.0.1:%d", port))
	gqlserver.Start()
	time.Sleep(100 * time.Millisecond)

	serverURI = fmt.Sprintf("http://127.0.0.1:%d/", port)

	os.Exit(m.Run())
}
