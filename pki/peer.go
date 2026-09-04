package pki

import (
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
)

type PeerCertificate x509.Certificate

func (pc *PeerCertificate) MarshalJSON() ([]byte, error) {
	b := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: pc.Raw})
	if b == nil {
		return nil, fmt.Errorf("failed to encode certificate to PEM")
	}
	return json.Marshal(b)
}

func (pc *PeerCertificate) UnmarshalJSON(b []byte) error {
	var buf []byte
	if err := json.Unmarshal(b, &buf); err != nil {
		return err
	}
	block, _ := pem.Decode(buf)
	if block == nil || block.Type != "CERTIFICATE" {
		return fmt.Errorf("peer certificate is not a valid CERTIFICATE PEM block")
	}
	c, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return err
	}
	*pc = PeerCertificate(*c)
	return nil
}
