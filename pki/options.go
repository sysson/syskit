package pki

import (
	"fmt"
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

// Options controls the contents of an Authority and the leaves it issues.
// The zero value is usable and yields ECDSA P-256 keys with the default
// validity periods; Organization and CACommonName are left blank if unset.
type Options struct {
	KeyType KeyType
	// RSABits is only consulted when KeyType is KeyTypeRSA.
	RSABits int

	// CADuration and Duration are the validity periods of the CA and of the
	// leaf certificates respectively.
	CADuration time.Duration
	Duration   time.Duration

	Organization string
	CACommonName string
}

type PKIOptions func(*Options)

func defaultOptions() Options {
	return Options{
		KeyType:      DefaultKeyType,
		RSABits:      DefaultRSABits,
		CADuration:   DefaultCADuration,
		Duration:     DefaultDuration,
		Organization: "",
		CACommonName: "",
	}
}

func WithKeyType(keyType KeyType) PKIOptions {
	return func(o *Options) {
		o.KeyType = keyType
	}
}

func WithRSABits(bits int) PKIOptions {
	return func(o *Options) {
		o.RSABits = bits
	}
}

func WithCADuration(d time.Duration) PKIOptions {
	return func(o *Options) {
		o.CADuration = d
	}
}

func WithDuration(d time.Duration) PKIOptions {
	return func(o *Options) {
		o.Duration = d
	}
}

func WithOrganization(org string) PKIOptions {
	return func(o *Options) {
		o.Organization = org
	}
}

func WithCACommonName(cn string) PKIOptions {
	return func(o *Options) {
		o.CACommonName = cn
	}
}

func (o *Options) Validate() error {
	switch o.KeyType {
	case KeyTypeECDSA, KeyTypeEd25519, KeyTypeRSA:
	default:
		return fmt.Errorf("unsupported key type %q: must be one of %s, %s, %s",
			o.KeyType, KeyTypeECDSA, KeyTypeEd25519, KeyTypeRSA)
	}
	if o.KeyType == KeyTypeRSA && o.RSABits < 2048 {
		return fmt.Errorf("rsa key size %d is too small: minimum is 2048", o.RSABits)
	}
	if o.CADuration <= 0 || o.Duration <= 0 {
		return fmt.Errorf("certificate durations must be positive")
	}
	return nil
}
