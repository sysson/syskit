package pki

import (
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"math/big"
	"time"
)

// Info summarizes a certificate for human-readable display.
type Info struct {
	CommonName   string
	SerialNumber *big.Int
	NotBefore    time.Time
	NotAfter     time.Time
	// Fingerprint is the hex-encoded SHA-256 digest of the DER certificate,
	// used to compare a local certificate against a copy held elsewhere.
	Fingerprint string
}

// ParseCertInfo parses a PEM-encoded certificate and summarizes it.
func ParseCertInfo(certPEM []byte) (Info, error) {
	block, _ := pem.Decode(certPEM)
	if block == nil || block.Type != blockCertificate {
		return Info{}, fmt.Errorf("not a valid %s PEM block", blockCertificate)
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return Info{}, fmt.Errorf("parsing certificate: %w", err)
	}
	sum := sha256.Sum256(cert.Raw)
	return Info{
		CommonName:   cert.Subject.CommonName,
		SerialNumber: cert.SerialNumber,
		NotBefore:    cert.NotBefore,
		NotAfter:     cert.NotAfter,
		Fingerprint:  fmt.Sprintf("%x", sum),
	}, nil
}
