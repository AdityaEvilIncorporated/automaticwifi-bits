// Package tlsconf builds the portal TLS config: OS trust store first, then an optional
// host-only SPKI pin (PROTOCOL.md section 6). Verification is never disabled.
package tlsconf

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"
)

// inspectTimeout bounds the handshake-only connection made by Inspect.
const inspectTimeout = 10 * time.Second

// SPKISHA256 returns lowercase hex SHA-256 of cert.RawSubjectPublicKeyInfo.
func SPKISHA256(cert *x509.Certificate) string {
	sum := sha256.Sum256(cert.RawSubjectPublicKeyInfo)
	return hex.EncodeToString(sum[:])
}

// VerifySystem verifies chain (leaf first) for host against roots (nil = OS trust store),
// using chain[1:] as intermediates, at the current time, for server auth.
func VerifySystem(chain []*x509.Certificate, host string, roots *x509.CertPool) error {
	if len(chain) == 0 || chain[0] == nil {
		return errors.New("tlsconf: empty certificate chain")
	}
	inter := x509.NewCertPool()
	for _, c := range chain[1:] {
		if c != nil {
			inter.AddCert(c)
		}
	}
	// With Roots nil, crypto/x509 uses the platform verifier (macOS SecTrust, Windows
	// CertVerify, the system bundle on Linux): that is the "OS trust store" of the protocol.
	_, err := chain[0].Verify(x509.VerifyOptions{
		DNSName:       host,
		Roots:         roots,
		Intermediates: inter,
		KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	})
	return err
}

// validPin reports whether pin is exactly 64 lowercase hex characters.
func validPin(pin string) bool {
	if len(pin) != 64 {
		return false
	}
	for i := 0; i < len(pin); i++ {
		c := pin[i]
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}

// PortalTLSConfig returns a *tls.Config for the portal client only:
//   - MinVersion TLS 1.2, ServerName = host.
//   - Standard verification (VerifySystem with roots) must pass, OR pin is non-empty and
//     SPKISHA256(leaf) == pin. Implemented with InsecureSkipVerify=true plus a
//     VerifyConnection callback that ALWAYS runs one of these checks and returns the original
//     verification error when neither passes; the callback also rejects a connection whose
//     ServerName differs from host (case-insensitive) so the config cannot be reused for
//     another host.
//
// pin must be "" or 64 lowercase hex chars, else an error. roots nil = OS trust store
// (tests pass the mock portal CA).
func PortalTLSConfig(host, pin string, roots *x509.CertPool) (*tls.Config, error) {
	if host == "" {
		return nil, errors.New("tlsconf: empty host")
	}
	if pin != "" && !validPin(pin) {
		return nil, errors.New("tlsconf: pinned_cert_sha256 must be 64 lowercase hex characters")
	}
	hostIsIP := net.ParseIP(host) != nil
	verify := func(cs tls.ConnectionState) error {
		// crypto/tls sends no SNI for an IP literal, so ConnectionState.ServerName is then "".
		// Accept that only when the configured host is an IP; a DNS host must match exactly.
		if !strings.EqualFold(cs.ServerName, host) && !(cs.ServerName == "" && hostIsIP) {
			return fmt.Errorf("tlsconf: portal TLS config for %s used for server name %q", host, cs.ServerName)
		}
		if len(cs.PeerCertificates) == 0 {
			return errors.New("tlsconf: server sent no certificate")
		}
		err := VerifySystem(cs.PeerCertificates, host, roots)
		if err == nil {
			return nil
		}
		if pin != "" && subtle.ConstantTimeCompare([]byte(SPKISHA256(cs.PeerCertificates[0])), []byte(pin)) == 1 {
			return nil
		}
		return err
	}
	// Semgrep's bypass-tls-verification rule is silenced: see the InsecureSkipVerify comment.
	return &tls.Config{ // nosemgrep
		MinVersion: tls.VersionTLS12,
		ServerName: host,
		// The built-in verifier is replaced, not disabled: VerifyConnection above runs on every
		// handshake (including resumed sessions) and fails closed. This is required to allow a
		// host-bound pin as a fallback, which crypto/tls cannot express otherwise.
		InsecureSkipVerify: true,
		VerifyConnection:   verify,
	}, nil
}

// Inspect does one TLS handshake to addr ("host:port") with SNI serverName and returns the
// peer certificate chain (leaf first) WITHOUT trusting it, then closes the connection. No
// application data is ever written on that connection; it exists only so the capture
// diagnostic can print the certificate fingerprint when verification fails. Timeout 10 s
// (via ctx or a dialer deadline, whichever is earlier).
func Inspect(ctx context.Context, addr, serverName string) ([]*x509.Certificate, error) {
	ctx, cancel := context.WithTimeout(ctx, inspectTimeout)
	defer cancel()
	// Semgrep's bypass-tls-verification rule is silenced: read-only handshake, see below.
	d := &tls.Dialer{Config: &tls.Config{ // nosemgrep
		MinVersion: tls.VersionTLS12,
		ServerName: serverName,
		// Read-only: the chain is returned for display and then checked with VerifySystem by the
		// caller. Nothing is sent on this connection, so trusting it is never required.
		InsecureSkipVerify: true,
	}}
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("tlsconf: handshake with %s: %w", addr, err)
	}
	defer conn.Close()
	tc, ok := conn.(*tls.Conn)
	if !ok {
		return nil, errors.New("tlsconf: not a TLS connection")
	}
	certs := tc.ConnectionState().PeerCertificates
	if len(certs) == 0 {
		return nil, errors.New("tlsconf: server sent no certificate")
	}
	return certs, nil
}
