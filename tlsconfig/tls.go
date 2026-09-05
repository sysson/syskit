package tlsconfig

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
func ServerTLSConfig(opts ...OptionFunc) (*tls.Config, error) {
	tlsCfg, err := defaultTLSConfig(opts...)
	if err != nil {
		return nil, err
	}
	if tlsCfg.ClientAuth >= tls.RequireAndVerifyClientCert && tlsCfg.caFile != nil {
		tlsCfg.ClientAuth = tls.RequireAndVerifyClientCert
		caPool, err := CertPool(tlsCfg.caFile)
		if err != nil {
			return nil, fmt.Errorf("loading CA pool: %w", err)
		}
		tlsCfg.ClientCAs = caPool
	}
	return &tlsCfg.Config, nil
}

// ClientTLSConfig builds a config presenting client and trusting caCert.
func ClientTLSConfig(opts ...OptionFunc) (*tls.Config, error) {
	tlsCfg, err := defaultTLSConfig(opts...)
	if err != nil {
		return nil, err
	}
	if !tlsCfg.InsecureSkipVerify && tlsCfg.caFile != nil {
		caPool, err := CertPool(tlsCfg.caFile)
		if err != nil {
			return nil, fmt.Errorf("loading CA pool: %w", err)
		}
		tlsCfg.RootCAs = caPool
	}
	return &tlsCfg.Config, nil
}
