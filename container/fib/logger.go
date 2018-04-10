package fib

import (
	"io"
	"io/ioutil"
	"log"
)

var logger = log.New(ioutil.Discard, "Fib ", log.LstdFlags)

func SetLogOutput(w io.Writer) {
	logger.SetOutput(w)
}
