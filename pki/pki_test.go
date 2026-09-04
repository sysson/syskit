package pki

import (
	"crypto/x509"
	"encoding/pem"
	"errors"
	"testing"
	"time"
)

func TestGenerateAndIssueVerifies(t *testing.T) {
	t.Parallel()

	for _, kt := range []KeyType{KeyTypeECDSA, KeyTypeEd25519, KeyTypeRSA} {
		t.Run(string(kt), func(t *testing.T) {
			t.Parallel()

			opts := Options{KeyType: kt, Organization: "test", CACommonName: "test-ca"}
			if kt == KeyTypeRSA {
				opts.RSABits = 2048
			}

			ca, err := NewAuthority(opts)
			if err != nil {
				t.Fatalf("NewAuthority: %v", err)
			}

			leaf, err := ca.Issue(LeafRequest{
				CommonName:  "leaf",
				ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
				DNSNames:    []string{"leaf.example.com"},
			})
			if err != nil {
				t.Fatalf("Issue: %v", err)
			}

			pool, err := CertPool(ca.KeyPair().Cert)
			if err != nil {
				t.Fatalf("CertPool: %v", err)
			}
			cert := parseCert(t, leaf.Cert)
			if _, err := cert.Verify(x509.VerifyOptions{
				Roots:     pool,
				DNSName:   "leaf.example.com",
				KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
			}); err != nil {
				t.Errorf("verifying leaf certificate: %v", err)
			}

			if _, err := ServerTLSConfig(leaf, ca.KeyPair().Cert); err != nil {
				t.Errorf("ServerTLSConfig: %v", err)
			}
			if _, err := ClientTLSConfig(leaf, ca.KeyPair().Cert); err != nil {
				t.Errorf("ClientTLSConfig: %v", err)
			}
		})
	}
}

func TestCAConstraints(t *testing.T) {
	t.Parallel()

	ca, err := NewAuthority(Options{})
	if err != nil {
		t.Fatalf("NewAuthority: %v", err)
	}
	cert := ca.Certificate()

	if !cert.IsCA {
		t.Error("CA certificate does not have IsCA set")
	}
	if !cert.BasicConstraintsValid {
		t.Error("CA certificate has invalid basic constraints")
	}
	if !cert.MaxPathLenZero {
		t.Error("CA certificate should have pathlen:0")
	}
	if cert.KeyUsage&x509.KeyUsageCertSign == 0 {
		t.Error("CA certificate cannot sign certificates")
	}
}

func TestReloadAuthorityIssuesTrustedLeaves(t *testing.T) {
	t.Parallel()

	opts := Options{}
	first, err := NewAuthority(opts)
	if err != nil {
		t.Fatalf("NewAuthority: %v", err)
	}

	reloaded, err := LoadAuthority(first.KeyPair(), opts)
	if err != nil {
		t.Fatalf("LoadAuthority: %v", err)
	}
	if !reloaded.Certificate().Equal(first.Certificate()) {
		t.Fatal("reloaded CA differs from the original")
	}

	leaf, err := reloaded.Issue(LeafRequest{
		CommonName:  "leaf",
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	})
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}

	pool, err := CertPool(first.KeyPair().Cert)
	if err != nil {
		t.Fatalf("CertPool: %v", err)
	}
	if _, err := parseCert(t, leaf.Cert).Verify(x509.VerifyOptions{
		Roots:     pool,
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}); err != nil {
		t.Errorf("leaf issued after reload not trusted by original CA: %v", err)
	}
}

func TestLeafKeyUsageByAlgorithm(t *testing.T) {
	t.Parallel()

	tests := []struct {
		keyType          KeyType
		wantEncipherment bool
	}{
		{KeyTypeECDSA, false},
		{KeyTypeEd25519, false},
		{KeyTypeRSA, true},
	}

	for _, tt := range tests {
		t.Run(string(tt.keyType), func(t *testing.T) {
			t.Parallel()

			opts := Options{KeyType: tt.keyType}
			if tt.keyType == KeyTypeRSA {
				opts.RSABits = 2048
			}
			ca, err := NewAuthority(opts)
			if err != nil {
				t.Fatalf("NewAuthority: %v", err)
			}
			leaf, err := ca.Issue(LeafRequest{
				CommonName:  "leaf",
				ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
			})
			if err != nil {
				t.Fatalf("Issue: %v", err)
			}

			cert := parseCert(t, leaf.Cert)
			if cert.KeyUsage&x509.KeyUsageDigitalSignature == 0 {
				t.Error("leaf certificate missing digitalSignature")
			}
			got := cert.KeyUsage&x509.KeyUsageKeyEncipherment != 0
			if got != tt.wantEncipherment {
				t.Errorf("keyEncipherment = %v, want %v", got, tt.wantEncipherment)
			}
		})
	}
}

func TestLeafNotAfterClampedToCA(t *testing.T) {
	t.Parallel()

	ca, err := NewAuthority(Options{CADuration: 24 * time.Hour, Duration: 365 * 24 * time.Hour})
	if err != nil {
		t.Fatalf("NewAuthority: %v", err)
	}
	leaf, err := ca.Issue(LeafRequest{
		CommonName:  "leaf",
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	})
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	if got := parseCert(t, leaf.Cert).NotAfter; got.After(ca.Certificate().NotAfter) {
		t.Errorf("leaf NotAfter %v outlives CA NotAfter %v", got, ca.Certificate().NotAfter)
	}
}

func TestOptionsValidation(t *testing.T) {
	t.Parallel()

	tests := map[string]Options{
		"unsupported key type": {KeyType: "dsa"},
		"weak rsa key":         {KeyType: KeyTypeRSA, RSABits: 512},
		"negative duration":    {Duration: -time.Hour},
	}

	for name, opts := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := NewAuthority(opts); err == nil {
				t.Fatal("expected an error, got nil")
			}
		})
	}
}

func TestIssueRequiresCommonNameAndKeyUsage(t *testing.T) {
	t.Parallel()

	ca, err := NewAuthority(Options{})
	if err != nil {
		t.Fatalf("NewAuthority: %v", err)
	}

	if _, err := ca.Issue(LeafRequest{ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}}); err == nil {
		t.Error("expected error for missing common name")
	}
	if _, err := ca.Issue(LeafRequest{CommonName: "leaf"}); err == nil {
		t.Error("expected error for missing ext key usage")
	}
}

func TestLoadAuthorityRejectsNonCA(t *testing.T) {
	t.Parallel()

	ca, err := NewAuthority(Options{})
	if err != nil {
		t.Fatalf("NewAuthority: %v", err)
	}
	leaf, err := ca.Issue(LeafRequest{
		CommonName:  "leaf",
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	})
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}

	_, err = LoadAuthority(KeyPair{Cert: leaf.Cert, Key: leaf.Key}, Options{})
	if err == nil {
		t.Fatal("expected an error loading a non-CA certificate as an authority")
	}
}

func TestErrNoAuthorityIsDistinguishable(t *testing.T) {
	t.Parallel()
	if !errors.Is(ErrNoAuthority, ErrNoAuthority) {
		t.Fatal("ErrNoAuthority should be comparable via errors.Is")
	}
}

func parseCert(t *testing.T, pemBytes []byte) *x509.Certificate {
	t.Helper()
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		t.Fatal("no PEM block found")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatalf("parsing certificate: %v", err)
	}
	return cert
}
