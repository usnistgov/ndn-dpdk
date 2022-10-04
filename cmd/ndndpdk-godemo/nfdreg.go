package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/chaseisabelle/flagz"
	"github.com/urfave/cli/v2"
	"github.com/usnistgov/ndn-dpdk/core/jsonhelper"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/endpoint"
	"github.com/usnistgov/ndn-dpdk/ndn/keychain"
	"github.com/usnistgov/ndn-dpdk/ndn/mgmt/nfdmgmt"
	"github.com/usnistgov/ndn-dpdk/ndn/tlv"
)

type nfdReg struct {
	Client        *nfdmgmt.Client
	ServedCerts   []ndn.Data
	DefaultOrigin int
	DefaultCost   int
	Commands      []nfdmgmt.ControlCommand
	CommandsDesc  []string
	Interval      time.Duration
	Repeat        time.Duration
}

func (cmd *nfdReg) readBase64File(filename string) (wire []byte, e error) {
	b64, e := os.ReadFile(filename)
	if e != nil {
		return nil, e
	}

	wire = make([]byte, base64.StdEncoding.DecodedLen(len(b64)))
	n, e := base64.StdEncoding.Decode(wire, b64)
	if e != nil {
		return nil, e
	}
	return wire[:n], nil
}

func (cmd *nfdReg) SetSigner(safeBagFile, safeBagPassphrase string) error {
	if safeBagFile == "" {
		return nil
	}

	wire, e := cmd.readBase64File(safeBagFile)
	if e != nil {
		return e
	}

	pvt, cert, e := keychain.ImportSafeBag(wire, []byte(safeBagPassphrase))
	if e != nil {
		return e
	}

	cmd.Client.Signer = pvt.WithKeyLocator(cert.Name())
	cmd.ServedCerts = append(cmd.ServedCerts, cert.Data())
	return nil
}

func (cmd *nfdReg) AddCert(certFile string) error {
	wire, e := cmd.readBase64File(certFile)
	if e != nil {
		return e
	}

	var pkt ndn.Packet
	if e := tlv.Decode(wire, &pkt); e != nil {
		return e
	}

	if pkt.Data == nil {
		return errors.New("not a Data")
	}

	cmd.ServedCerts = append(cmd.ServedCerts, *pkt.Data)
	return nil
}

func (cmd *nfdReg) AddRegisterCommand(p string) (e error) {
	var c nfdmgmt.RibRegisterCommand
	if strings.HasPrefix(p, "/") {
		e = jsonhelper.Roundtrip(map[string]any{
			"name":      p,
			"origin":    cmd.DefaultOrigin,
			"cost":      cmd.DefaultCost,
			"noInherit": true,
			"capture":   true,
		}, &c)
	} else {
		e = json.Unmarshal([]byte(p), &c)
	}
	if e != nil {
		return e
	}

	cmd.addCommand(c)
	return nil
}

func (cmd *nfdReg) AddUnregisterCommand(p string) (e error) {
	var c nfdmgmt.RibUnregisterCommand
	if strings.HasPrefix(p, "/") {
		e = jsonhelper.Roundtrip(map[string]any{
			"name":   p,
			"origin": cmd.DefaultOrigin,
			"cost":   cmd.DefaultCost,
		}, &c)
	} else {
		e = json.Unmarshal([]byte(p), &c)
	}
	if e != nil {
		return e
	}

	cmd.addCommand(c)
	return nil
}

func (cmd *nfdReg) addCommand(c nfdmgmt.ControlCommand) {
	cmd.Commands = append(cmd.Commands, c)

	var b bytes.Buffer
	fmt.Fprintf(&b, "%T ", c)
	json.NewEncoder(&b).Encode(c)
	cmd.CommandsDesc = append(cmd.CommandsDesc, string(bytes.TrimRight(b.Bytes(), "\n")))
}

func (cmd *nfdReg) StartServeCerts(ctx context.Context) {
	for _, cert := range cmd.ServedCerts {
		cert := cert
		endpoint.Produce(ctx, endpoint.ProducerOptions{
			Prefix:  keychain.ToKeyName(cert.Name),
			Handler: func(ctx context.Context, interest ndn.Interest) (ndn.Data, error) { return cert, nil },
		})
	}
}

func (cmd *nfdReg) SendCommands(ctx context.Context) {
	for i, command := range cmd.Commands {
		cr, e := cmd.Client.Invoke(ctx, command)
		if e != nil {
			log.Printf("%v %s", e, cmd.CommandsDesc[i])
		} else {
			log.Printf("%d %s", cr.StatusCode, cmd.CommandsDesc[i])
		}
		time.Sleep(cmd.Interval)
	}
}

func init() {
	var commandPrefixURI, safeBagFile, safeBagPassphrase string
	var serveCertz, registerz, unregisterz flagz.Flagz
	var cmd nfdReg

	defineCommand(&cli.Command{
		Name:  "nfdreg",
		Usage: "Register a prefix on NFD uplink.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "command",
				Usage:       "NFD command prefix.",
				Value:       nfdmgmt.PrefixLocalhop.String(),
				Destination: &commandPrefixURI,
			},
			&cli.StringFlag{
				Name:        "signer",
				Usage:       "Signer key SafeBag file.",
				Destination: &safeBagFile,
			},
			&cli.StringFlag{
				Name:        "signer-pass",
				Usage:       "Signer key SafeBag passphrase.",
				Destination: &safeBagPassphrase,
			},
			&cli.GenericFlag{
				Name:  "serve-cert",
				Usage: "Serve certificate(s).",
				Value: &serveCertz,
			},
			&cli.IntFlag{
				Name:        "origin",
				Usage:       "Route origin, for prefix(es) not specified as JSON.",
				Value:       nfdmgmt.RouteOriginClient,
				Destination: &cmd.DefaultOrigin,
			},
			&cli.IntFlag{
				Name:        "cost",
				Usage:       "Route cost, for prefix(es) not specified as JSON.",
				Value:       0,
				Destination: &cmd.DefaultCost,
			},
			&cli.GenericFlag{
				Name:  "register",
				Usage: "Register prefix(es).",
				Value: &registerz,
			},
			&cli.GenericFlag{
				Name:  "unregister",
				Usage: "Unregister prefix(es).",
				Value: &unregisterz,
			},
			&cli.DurationFlag{
				Name:        "interval",
				Usage:       "Interval between commands.",
				Value:       1 * time.Second,
				Destination: &cmd.Interval,
			},
			&cli.DurationFlag{
				Name:        "repeat",
				Usage:       "Repeat process after this duration. If zero, run once and exit.",
				Destination: &cmd.Repeat,
			},
		},
		Before: func(c *cli.Context) error {
			client, e := nfdmgmt.New()
			if e != nil {
				return e
			}
			cmd.Client = client
			cmd.Client.Prefix = ndn.ParseName(commandPrefixURI)

			if e := cmd.SetSigner(safeBagFile, safeBagPassphrase); e != nil {
				return fmt.Errorf("import signer SafeBag: %w", e)
			}
			for i, certFile := range serveCertz.Array() {
				if e := cmd.AddCert(certFile); e != nil {
					return fmt.Errorf("add cert %d: %w", i, e)
				}
			}

			for i, p := range unregisterz.Array() {
				if e := cmd.AddUnregisterCommand(p); e != nil {
					return fmt.Errorf("unregister command %d: %w", i, e)
				}
			}
			for i, p := range registerz.Array() {
				if e := cmd.AddRegisterCommand(p); e != nil {
					return fmt.Errorf("register command %d: %w", i, e)
				}
			}

			return openUplink(c)
		},
		Action: func(c *cli.Context) error {
			cmd.StartServeCerts(c.Context)
			for {
				cmd.SendCommands(c.Context)
				if cmd.Repeat <= 0 {
					return nil
				}
				select {
				case <-c.Context.Done():
					return c.Context.Err()
				case <-time.After(cmd.Repeat):
				}
			}
		},
	})
}
