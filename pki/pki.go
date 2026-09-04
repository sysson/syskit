// Package pki is a small self-signed certificate authority toolkit: it
// issues a CA and mTLS leaf certificates, performing no I/O beyond reading
// the system entropy source. Everything is returned as PEM blocks so callers
// can persist them however they like (local files, Kubernetes Secrets, ...).
package pki

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net"
	"time"
)

// ErrNoAuthority is returned by callers loading a CA that isn't present in
// their backing store.
var ErrNoAuthority = errors.New("no certificate authority found")

// KeyPair is a PEM-encoded certificate and its private key.
type KeyPair struct {
	Cert []byte
	Key  []byte
}

// Authority is a certificate authority capable of issuing leaf certificates.
type Authority struct {
	cert *x509.Certificate
	key  crypto.Signer
	pair KeyPair
	opts Options
}

// KeyPair returns the PEM-encoded CA certificate and key. The key must never
// leave a trusted store; only the certificate is needed to verify peers.
func (a *Authority) KeyPair() KeyPair { return a.pair }

// Certificate returns the parsed CA certificate.
func (a *Authority) Certificate() *x509.Certificate { return a.cert }

// NewAuthority creates a new self-signed certificate authority.
func NewAuthority(opts ...PKIOptions) (*Authority, error) {
	o := defaultOptions()
	for _, opt := range opts {
		opt(&o)
	}
	if err := o.Validate(); err != nil {
		return nil, err
	}

	key, err := generateKey(o.KeyType, o.RSABits)
	if err != nil {
		return nil, fmt.Errorf("generating CA key: %w", err)
	}

	serial, err := newSerial()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	tmpl := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			Organization: []string{o.Organization},
			CommonName:   o.CACommonName,
		},
		NotBefore:             now.Add(-backdate),
		NotAfter:              now.Add(o.CADuration),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            0,
		MaxPathLenZero:        true,
	}

	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, key.Public(), key)
	if err != nil {
		return nil, fmt.Errorf("signing CA certificate: %w", err)
	}

	cert, err := x509.ParseCertificate(der)
	if err != nil {
		return nil, fmt.Errorf("parsing generated CA certificate: %w", err)
	}

	keyPEM, err := encodeKey(key)
	if err != nil {
		return nil, err
	}

	return &Authority{
		cert: cert,
		key:  key,
		pair: KeyPair{Cert: encodePEM(blockCertificate, der), Key: keyPEM},
		opts: o,
	}, nil
}

// LoadAuthority restores an Authority from previously generated PEM blocks so
// that new leaves can be issued without rotating the CA.
func LoadAuthority(pair KeyPair) (*Authority, error) {

	block, _ := pem.Decode(pair.Cert)
	if block == nil || block.Type != blockCertificate {
		return nil, fmt.Errorf("CA certificate is not a valid %s PEM block", blockCertificate)
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parsing CA certificate: %w", err)
	}
	if !cert.IsCA {
		return nil, fmt.Errorf("certificate %q is not a CA", cert.Subject.CommonName)
	}

	block, _ = pem.Decode(pair.Key)
	if block == nil || block.Type != blockPrivateKey {
		return nil, fmt.Errorf("CA key is not a valid %s PEM block", blockPrivateKey)
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parsing CA key: %w", err)
	}
	key, ok := parsed.(crypto.Signer)
	if !ok {
		return nil, fmt.Errorf("CA key of type %T cannot sign", parsed)
	}
	o := defaultOptions()
	if len(cert.Subject.Organization) > 0 {
		o.Organization = cert.Subject.Organization[0]
	}
	o.CACommonName = cert.Subject.CommonName
	o.CADuration = cert.NotAfter.Sub(cert.NotBefore)

	switch key := key.(type) {
	case *ecdsa.PrivateKey:
		o.KeyType = KeyTypeECDSA
	case ed25519.PrivateKey:
		o.KeyType = KeyTypeEd25519
	case *rsa.PrivateKey:
		o.KeyType = KeyTypeRSA
		o.RSABits = key.N.BitLen()
	default:
		return nil, fmt.Errorf("unsupported CA key type %T", key)
	}
	return &Authority{cert: cert, key: key, pair: pair, opts: o}, nil
}

// LeafRequest describes a certificate to be issued by an Authority.
type LeafRequest struct {
	CommonName string
	// Organization overrides the Authority's default organization.
	Organization string
	ExtKeyUsage  []x509.ExtKeyUsage
	DNSNames     []string
	IPAddresses  []net.IP
}

// Issue signs a new leaf certificate with a freshly generated key.
func (a *Authority) Issue(req LeafRequest) (KeyPair, error) {
	if req.CommonName == "" {
		return KeyPair{}, fmt.Errorf("leaf certificate common name is required")
	}
	if len(req.ExtKeyUsage) == 0 {
		return KeyPair{}, fmt.Errorf("leaf certificate %q needs at least one extended key usage", req.CommonName)
	}

	key, err := generateKey(a.opts.KeyType, a.opts.RSABits)
	if err != nil {
		return KeyPair{}, fmt.Errorf("generating key for %q: %w", req.CommonName, err)
	}

	serial, err := newSerial()
	if err != nil {
		return KeyPair{}, err
	}

	// keyEncipherment is meaningless for ECDSA/Ed25519 and is rejected by some
	// verifiers, so it is only set for RSA where RSA key transport is possible.
	keyUsage := x509.KeyUsageDigitalSignature
	if a.opts.KeyType == KeyTypeRSA {
		keyUsage |= x509.KeyUsageKeyEncipherment
	}

	now := time.Now()
	notAfter := now.Add(a.opts.Duration)
	if notAfter.After(a.cert.NotAfter) {
		notAfter = a.cert.NotAfter
	}

	organization := a.opts.Organization
	if req.Organization != "" {
		organization = req.Organization
	}

	tmpl := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			Organization: []string{organization},
			CommonName:   req.CommonName,
		},
		NotBefore:             now.Add(-backdate),
		NotAfter:              notAfter,
		KeyUsage:              keyUsage,
		ExtKeyUsage:           req.ExtKeyUsage,
		BasicConstraintsValid: true,
		IsCA:                  false,
		DNSNames:              req.DNSNames,
		IPAddresses:           req.IPAddresses,
	}

	der, err := x509.CreateCertificate(rand.Reader, tmpl, a.cert, key.Public(), a.key)
	if err != nil {
		return KeyPair{}, fmt.Errorf("signing certificate for %q: %w", req.CommonName, err)
	}

	keyPEM, err := encodeKey(key)
	if err != nil {
		return KeyPair{}, err
	}

	return KeyPair{Cert: encodePEM(blockCertificate, der), Key: keyPEM}, nil
}

func generateKey(kt KeyType, rsaBits int) (crypto.Signer, error) {
	switch kt {
	case KeyTypeECDSA:
		return ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	case KeyTypeEd25519:
		_, key, err := ed25519.GenerateKey(rand.Reader)
		return key, err
	case KeyTypeRSA:
		return rsa.GenerateKey(rand.Reader, rsaBits)
	default:
		return nil, fmt.Errorf("unsupported key type %q", kt)
	}
}

func newSerial() (*big.Int, error) {
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, fmt.Errorf("generating certificate serial number: %w", err)
	}
	return serial, nil
}

func encodeKey(key crypto.Signer) ([]byte, error) {
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("marshalling private key: %w", err)
	}
	return encodePEM(blockPrivateKey, der), nil
}

func encodePEM(blockType string, der []byte) []byte {
	return pem.EncodeToMemory(&pem.Block{Type: blockType, Bytes: der})
}
