package yamlflag_test

import (
	"flag"
	"io/ioutil"
	"os"
	"testing"

	"github.com/usnistgov/ndn-dpdk/core/yamlflag"
)

type Document struct {
	A int
	B string
}

func TestNewNonPtr(t *testing.T) {
	assert, _ := makeAR(t)

	var doc Document
	assert.Panics(func() { yamlflag.New(doc) })
	assert.Panics(func() { yamlflag.New(1) })
}

func TestYaml(t *testing.T) {
	assert, _ := makeAR(t)

	flags := flag.NewFlagSet("", flag.PanicOnError)
	var doc Document
	flags.Var(yamlflag.New(&doc), "var", "")
	flags.Parse([]string{"--var=A: 1\nB: bb"})

	assert.Equal(1, doc.A)
	assert.Equal("bb", doc.B)
}

func TestJson(t *testing.T) {
	assert, _ := makeAR(t)

	flags := flag.NewFlagSet("", flag.PanicOnError)
	var doc Document
	flags.Var(yamlflag.New(&doc), "var", "")
	flags.Parse([]string{"--var={\"A\":1,\"B\":\"bb\"}"})

	assert.Equal(1, doc.A)
	assert.Equal("bb", doc.B)
}

func TestFile(t *testing.T) {
	assert, require := makeAR(t)

	file, e := ioutil.TempFile("", "yamlflag-test")
	require.NoError(e)
	filename := file.Name()
	defer os.Remove(filename)
	_, e = file.WriteString("A: 1\nB: bb\n")
	require.NoError(e)
	require.NoError(file.Close())

	flags := flag.NewFlagSet("", flag.PanicOnError)
	var doc Document
	flags.Var(yamlflag.New(&doc), "var", "")
	flags.Parse([]string{"--var=@" + filename})

	assert.Equal(1, doc.A)
	assert.Equal("bb", doc.B)
}
