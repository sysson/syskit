package tlsconfig

import (
	"crypto/tls"
	"fmt"
)

type options struct {
	tls.Config
	certFile []byte
	keyFile  []byte
	caFile   []byte
}

type OptionFunc func(*options)

func defaultTLSConfig(optionFuncs ...OptionFunc) (*options, error) {
	opts := options{
		MinVersion: tls.VersionTLS12,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		},
	}
	for _, opt := range optionFuncs {
		opt(&opts)
	}
	cert, err := tls.X509KeyPair(opts.certFile, opts.keyFile)
	if err != nil {
		return nil, fmt.Errorf("loading keypair: %w", err)
	}
	opts.Certificates = []tls.Certificate{cert}
	if err := checkValidVersion(opts.MinVersion); err != nil {
		return nil, err
	}
	if opts.MinVersion < tls.VersionTLS13 && len(opts.CipherSuites) == 0 {
		return nil, fmt.Errorf("cipher suites must be specified for TLS versions below 1.3")
	}
	return &opts, nil
}

func WithKeyPair(cert, key []byte) OptionFunc {
	return func(opts *options) {
		opts.certFile = cert
		opts.keyFile = key
	}
}

func WithInsecureSkipVerify(skip bool) OptionFunc {
	return func(opts *options) {
		opts.InsecureSkipVerify = skip
	}
}

func WithCipherSuites(suites []uint16) OptionFunc {
	return func(opts *options) {
		opts.CipherSuites = suites
	}
}

func WithCA(ca []byte) OptionFunc {
	return func(opts *options) {
		opts.caFile = ca
	}
}

func WithClientAuth(auth tls.ClientAuthType) OptionFunc {
	return func(opts *options) {
		opts.ClientAuth = auth
	}
}

func WithNextProtos(protos []string) OptionFunc {
	return func(opts *options) {
		opts.NextProtos = protos
	}
}

func WithMinVersion(version uint16) OptionFunc {
	return func(opts *options) {
		opts.MinVersion = version
	}
}

func checkValidVersion(version uint16) error {
	switch version {
	case tls.VersionTLS10, tls.VersionTLS11, tls.VersionTLS12, tls.VersionTLS13:
		return nil
	default:
		return fmt.Errorf("invalid TLS version %d: must be one of %d, %d, %d, %d",
			version, tls.VersionTLS10, tls.VersionTLS11, tls.VersionTLS12, tls.VersionTLS13)
	}
}
