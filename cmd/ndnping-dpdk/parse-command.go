package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"ndn-dpdk/iface/faceuri"
)

type parsedCommand struct {
	servers        []serverCfg
	measureLatency bool
	measureRtt     bool
	addDelay       time.Duration
	serverNack     bool
}

type serverCfg struct {
	face faceuri.FaceUri
}

func parseCommand(args []string) (pc parsedCommand, e error) {
	flags := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	flags.BoolVar(&pc.measureLatency, "latency", false, "measure client-server latency")
	flags.BoolVar(&pc.measureRtt, "rtt", false, "measure round trip time")
	flags.DurationVar(&pc.addDelay, "add-delay", time.Duration(0), "add delay before server response")
	flags.BoolVar(&pc.serverNack, "nack", true, "server Nacks on unserved Interests")

	e = flags.Parse(args)
	if e != nil {
		return pc, e
	}

	const (
		STATE_NONE          = iota
		STATE_CLIENT_FACE   // next token is client face
		STATE_CLIENT_PREFIX // next token is client prefix
		STATE_CLIENT_PCT    // next token is client percentage
		STATE_SERVER_FACE   // next token is server face
		STATE_SERVER_PREFIX // next token is server prefix or end of server defintion
	)
	state := STATE_NONE
	isIdleState := func() bool { // can accept +c +s or end?
		switch state {
		case STATE_NONE, STATE_CLIENT_PREFIX, STATE_SERVER_PREFIX:
			return true
		}
		return false
	}
	parseToken := func(token string) error {
		switch {
		case isIdleState():
			switch token {
			case "+c":
				state = STATE_CLIENT_FACE
			case "+s":
				state = STATE_SERVER_FACE
			}
		case state == STATE_CLIENT_FACE:
			state = STATE_CLIENT_PREFIX
		case state == STATE_CLIENT_PREFIX:
			state = STATE_CLIENT_PCT
		case state == STATE_CLIENT_PCT:
		case state == STATE_SERVER_FACE:
			u, e := faceuri.Parse(token)
			if e != nil {
				return e
			}
			pc.servers = append(pc.servers, serverCfg{*u})
			state = STATE_SERVER_PREFIX
		case state == STATE_SERVER_PREFIX:
		}
		return nil
	}
	for _, token := range flags.Args() {
		e = parseToken(token)
		if e != nil {
			return pc, fmt.Errorf("command line error near %s: %v", token, e)
		}
	}
	if !isIdleState() {
		return pc, fmt.Errorf("command line is incomplete (state=%d)", state)
	}

	return pc, nil
}
