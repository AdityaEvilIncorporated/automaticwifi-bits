// Package tlsconf builds the portal TLS config: OS trust store first, then an optional
// host-only SPKI pin (PROTOCOL.md section 6). Verification is never disabled.
package tlsconf

import (
	"context"
	"crypto/tls"
	"crypto/x509"
)

// SPKISHA256 returns lowercase hex SHA-256 of cert.RawSubjectPublicKeyInfo.
func SPKISHA256(cert *x509.Certificate) string {
	panic("not implemented")
}

// VerifySystem verifies chain (leaf first) for host against roots (nil = OS trust store),
// using chain[1:] as intermediates, at the current time, for server auth.
func VerifySystem(chain []*x509.Certificate, host string, roots *x509.CertPool) error {
	panic("not implemented")
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
	panic("not implemented")
}

// Inspect does one TLS handshake to addr ("host:port") with SNI serverName and returns the
// peer certificate chain (leaf first) WITHOUT trusting it, then closes the connection. No
// application data is ever written on that connection; it exists only so the capture
// diagnostic can print the certificate fingerprint when verification fails. Timeout 10 s
// (via ctx or a dialer deadline, whichever is earlier).
func Inspect(ctx context.Context, addr, serverName string) ([]*x509.Certificate, error) {
	panic("not implemented")
}
