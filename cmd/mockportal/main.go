// Command mockportal runs the local FortiGate stand-in (internal/mockportal) for manual tests
// and for the Kotlin test suite. It prints one JSON line with the config once listening, then
// serves until killed.
//
//	mockportal -probe 127.0.0.1:18080 -portal 127.0.0.1:18443 -ca-out /tmp/mock-ca.pem
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"autowifi/internal/mockportal"
	"autowifi/internal/tlsconf"
)

func main() {
	probe := flag.String("probe", "127.0.0.1:0", "probe server listen address")
	portal := flag.String("portal", "127.0.0.1:0", "portal (TLS) listen address")
	caOut := flag.String("ca-out", "", "write the portal certificate (PEM) to this file")
	user := flag.String("username", mockportal.DefaultUsername, "accepted username")
	pass := flag.String("password", mockportal.DefaultPassword, "accepted password")
	flag.Parse()

	s := mockportal.Start(mockportal.Options{Username: *user, Password: *pass, ProbeAddr: *probe, PortalAddr: *portal})
	defer s.Close()
	if *caOut != "" {
		if err := os.WriteFile(*caOut, s.CertPEM(), 0o644); err != nil {
			fmt.Fprintln(os.Stderr, "mockportal:", err)
			os.Exit(1)
		}
	}
	cfg := s.Config()
	// Go's SPKI pin of the mock certificate, so the Kotlin tests can check they compute the same.
	cfg.PinnedCertSHA256 = tlsconf.SPKISHA256(s.Portal.Certificate())
	_ = json.NewEncoder(os.Stdout).Encode(cfg)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig
}
