package pki

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
)

// CertPool returns a pool trusting caCert.
func CertPool(caCert []byte) (*x509.CertPool, error) {
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("CA certificate contains no usable PEM block")
	}
	return pool, nil
}

// ServerTLSConfig builds a config requiring and verifying client certificates
// issued by caCert.
func ServerTLSConfig(server KeyPair, caCert []byte) (*tls.Config, error) {
	cert, err := tls.X509KeyPair(server.Cert, server.Key)
	if err != nil {
		return nil, fmt.Errorf("loading server keypair: %w", err)
	}
	pool, err := CertPool(caCert)
	if err != nil {
		return nil, err
	}
	return &tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{cert},
		ClientCAs:    pool,
		ClientAuth:   tls.RequireAndVerifyClientCert,
		NextProtos:   []string{"h2", "http/1.1"},
	}, nil
}

// ClientTLSConfig builds a config presenting client and trusting caCert.
func ClientTLSConfig(client KeyPair, caCert []byte) (*tls.Config, error) {
	cert, err := tls.X509KeyPair(client.Cert, client.Key)
	if err != nil {
		return nil, fmt.Errorf("loading client keypair: %w", err)
	}
	pool, err := CertPool(caCert)
	if err != nil {
		return nil, err
	}
	return &tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{cert},
		RootCAs:      pool,
	}, nil
}
