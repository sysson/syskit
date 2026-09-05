package pki

import (
	"crypto/x509"
	"fmt"
	"net"
	"time"
)

// KeyType selects the public key algorithm used for every certificate in a
// bundle. The CA and its leaves always share the same algorithm.
type KeyType string

const (
	KeyTypeECDSA   KeyType = "ecdsa"
	KeyTypeEd25519 KeyType = "ed25519"
	KeyTypeRSA     KeyType = "rsa"
)

const (
	DefaultKeyType    = KeyTypeECDSA
	DefaultRSABits    = 3072
	DefaultCADuration = 3650 * 24 * time.Hour
	DefaultDuration   = 365 * 24 * time.Hour

	blockCertificate = "CERTIFICATE"
	blockPrivateKey  = "PRIVATE KEY"

	// Backdate to absorb clock skew between the issuing machine and the peer
	// verifying the certificate.
	backdate = time.Hour
)

// Options controls the contents of a Certificate request
// and the resulting certificate.
type Options struct {
	KeyType KeyType
	// RSABits is only consulted when KeyType is KeyTypeRSA.
	RSABits int

	// CADuration is the validity period of the certificate.
	Duration time.Duration

	// Organization is the organization name to use in the certificate subject.
	// CommonName is the common name to use in the certificate subject.
	Organization string
	CommonName   string

	// Leaf certificate options
	ExtKeyUsage []x509.ExtKeyUsage
	DNSNames    []string
	IPAddresses []net.IP
}

type OptionFunc func(*Options)

func defaultOptions(opts ...OptionFunc) Options {
	o := Options{
		KeyType:  DefaultKeyType,
		RSABits:  DefaultRSABits,
		Duration: DefaultDuration,
	}
	for _, opt := range opts {
		opt(&o)
	}
	return o
}

func WithKeyType(keyType KeyType) OptionFunc {
	return func(o *Options) {
		o.KeyType = keyType
	}
}

func WithRSABits(bits int) OptionFunc {
	return func(o *Options) {
		o.RSABits = bits
	}
}

func WithDuration(d time.Duration) OptionFunc {
	return func(o *Options) {
		o.Duration = d
	}
}

func WithOrganization(org string) OptionFunc {
	return func(o *Options) {
		o.Organization = org
	}
}

func WithCommonName(cn string) OptionFunc {
	return func(o *Options) {
		o.CommonName = cn
	}
}

func WithExtKeyUsage(eku []x509.ExtKeyUsage) OptionFunc {
	return func(o *Options) {
		o.ExtKeyUsage = eku
	}
}

func WithDNSNames(dns []string) OptionFunc {
	return func(o *Options) {
		o.DNSNames = dns
	}
}

func WithIPAddresses(ips []net.IP) OptionFunc {
	return func(o *Options) {
		o.IPAddresses = ips
	}
}

func (o *Options) validate() error {
	switch o.KeyType {
	case KeyTypeECDSA, KeyTypeEd25519, KeyTypeRSA:
	default:
		return fmt.Errorf("unsupported key type %q: must be one of %s, %s, %s",
			o.KeyType, KeyTypeECDSA, KeyTypeEd25519, KeyTypeRSA)
	}
	if o.KeyType == KeyTypeRSA && o.RSABits < 2048 {
		return fmt.Errorf("rsa key size %d is too small: minimum is 2048", o.RSABits)
	}
	if o.Duration <= 0 {
		return fmt.Errorf("certificate duration must be positive")
	}
	return nil
}

func (o *Options) validateLeafRequest() error {
	err := o.validate()
	if err != nil {
		return err
	}
	if o.CommonName == "" {
		return fmt.Errorf("leaf certificate common name is required")
	}
	if len(o.ExtKeyUsage) == 0 {
		return fmt.Errorf("leaf certificate %q needs at least one extended key usage", o.CommonName)
	}
	return nil
}