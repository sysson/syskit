package tlsconfig_test

import (
	"crypto/tls"
	"crypto/x509"
	"net"
	"testing"

	"github.com/sysson/syskit/pki"
	"github.com/sysson/syskit/tlsconfig"
)

func TestCertPool(t *testing.T) {
	t.Parallel()

	ca := newAuthority(t)
	pool, err := tlsconfig.CertPool(ca.KeyPair().Cert)
	if err != nil {
		t.Fatalf("CertPool: %v", err)
	}
	want := x509.NewCertPool()
	if !want.AppendCertsFromPEM(ca.KeyPair().Cert) {
		t.Fatal("failed to build expected certificate pool")
	}
	if !pool.Equal(want) {
		t.Fatal("CertPool returned a different certificate pool")
	}
	if _, err := tlsconfig.CertPool([]byte("not a certificate")); err == nil {
		t.Fatal("CertPool accepted invalid PEM")
	}
}

func TestServerTLSConfig(t *testing.T) {
	t.Parallel()

	ca := newAuthority(t)
	server := issue(t, ca, "server", x509.ExtKeyUsageServerAuth)
	got, err := tlsconfig.ServerTLSConfig(
		tlsconfig.WithKeyPair(server.Cert, server.Key),
		tlsconfig.WithCA(ca.KeyPair().Cert),
		tlsconfig.WithClientAuth(tls.RequireAndVerifyClientCert),
		tlsconfig.WithNextProtos([]string{"h2"}),
	)
	if err != nil {
		t.Fatalf("ServerTLSConfig: %v", err)
	}
	if got.MinVersion != tls.VersionTLS12 {
		t.Errorf("MinVersion = %d, want %d", got.MinVersion, tls.VersionTLS12)
	}
	if got.ClientAuth != tls.RequireAndVerifyClientCert {
		t.Errorf("ClientAuth = %d, want %d", got.ClientAuth, tls.RequireAndVerifyClientCert)
	}
	if got.ClientCAs == nil || !got.ClientCAs.Equal(wantPool(t, ca.KeyPair().Cert)) {
		t.Fatal("ClientCAs does not contain the configured CA")
	}
	if len(got.Certificates) != 1 {
		t.Fatalf("Certificates = %d, want 1", len(got.Certificates))
	}
	if len(got.NextProtos) != 1 || got.NextProtos[0] != "h2" {
		t.Fatalf("NextProtos = %v, want [h2]", got.NextProtos)
	}
}

func TestClientTLSConfig(t *testing.T) {
	t.Parallel()

	ca := newAuthority(t)
	client := issue(t, ca, "client", x509.ExtKeyUsageClientAuth)
	got, err := tlsconfig.ClientTLSConfig(
		tlsconfig.WithKeyPair(client.Cert, client.Key),
		tlsconfig.WithCA(ca.KeyPair().Cert),
		tlsconfig.WithMinVersion(tls.VersionTLS13),
	)
	if err != nil {
		t.Fatalf("ClientTLSConfig: %v", err)
	}
	if got.InsecureSkipVerify {
		t.Fatal("InsecureSkipVerify = true, want false")
	}
	if got.MinVersion != tls.VersionTLS13 {
		t.Errorf("MinVersion = %d, want %d", got.MinVersion, tls.VersionTLS13)
	}
	if got.RootCAs == nil || !got.RootCAs.Equal(wantPool(t, ca.KeyPair().Cert)) {
		t.Fatal("RootCAs does not contain the configured CA")
	}
	if len(got.Certificates) != 1 {
		t.Fatalf("Certificates = %d, want 1", len(got.Certificates))
	}
}

func TestTLSConfigValidation(t *testing.T) {
	t.Parallel()

	ca := newAuthority(t)
	leaf := issue(t, ca, "leaf", x509.ExtKeyUsageServerAuth)
	valid := tlsconfig.WithKeyPair(leaf.Cert, leaf.Key)
	tests := []struct {
		name string
		opts []tlsconfig.OptionFunc
	}{
		{name: "invalid keypair", opts: []tlsconfig.OptionFunc{tlsconfig.WithKeyPair([]byte("bad"), []byte("bad"))}},
		{name: "invalid TLS version", opts: []tlsconfig.OptionFunc{valid, tlsconfig.WithMinVersion(99)}},
		{name: "empty TLS 1.2 cipher suites", opts: []tlsconfig.OptionFunc{valid, tlsconfig.WithCipherSuites(nil)}},
		{name: "invalid CA", opts: []tlsconfig.OptionFunc{valid, tlsconfig.WithCA([]byte("bad")), tlsconfig.WithClientAuth(tls.RequireAndVerifyClientCert)}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := tlsconfig.ServerTLSConfig(tt.opts...); err == nil {
				t.Fatal("ServerTLSConfig succeeded unexpectedly")
			}
			if _, err := tlsconfig.ClientTLSConfig(tt.opts...); err == nil {
				t.Fatal("ClientTLSConfig succeeded unexpectedly")
			}
		})
	}
}

func TestMutualTLS(t *testing.T) {
	t.Parallel()

	ca := newAuthority(t)
	server := issue(t, ca, "server", x509.ExtKeyUsageServerAuth)
	client := issue(t, ca, "client", x509.ExtKeyUsageClientAuth)
	serverConfig, err := tlsconfig.ServerTLSConfig(
		tlsconfig.WithKeyPair(server.Cert, server.Key),
		tlsconfig.WithCA(ca.KeyPair().Cert),
		tlsconfig.WithClientAuth(tls.RequireAndVerifyClientCert),
	)
	if err != nil {
		t.Fatalf("ServerTLSConfig: %v", err)
	}
	clientConfig, err := tlsconfig.ClientTLSConfig(
		tlsconfig.WithKeyPair(client.Cert, client.Key),
		tlsconfig.WithCA(ca.KeyPair().Cert),
		tlsconfig.WithInsecureSkipVerify(true),
	)
	if err != nil {
		t.Fatalf("ClientTLSConfig: %v", err)
	}

	serverConn, clientConn := net.Pipe()
	serverTLS := tls.Server(serverConn, serverConfig)
	clientTLS := tls.Client(clientConn, clientConfig)
	serverErr := make(chan error, 1)
	go func() {
		serverErr <- serverTLS.Handshake()
	}()
	if err := clientTLS.Handshake(); err != nil {
		t.Fatalf("client handshake: %v", err)
	}
	if err := <-serverErr; err != nil {
		t.Fatalf("server handshake: %v", err)
	}
	_ = serverTLS.Close()
	_ = clientTLS.Close()
}

func newAuthority(t *testing.T) *pki.Authority {
	t.Helper()
	ca, err := pki.NewAuthority()
	if err != nil {
		t.Fatalf("NewAuthority: %v", err)
	}
	return ca
}

func issue(t *testing.T, ca *pki.Authority, commonName string, usage x509.ExtKeyUsage) pki.KeyPair {
	t.Helper()
	leaf, err := ca.Issue(
		pki.WithCommonName(commonName),
		pki.WithExtKeyUsage([]x509.ExtKeyUsage{usage}),
	)
	if err != nil {
		t.Fatalf("Issue %s certificate: %v", commonName, err)
	}
	return leaf
}

func wantPool(t *testing.T, certPEM []byte) *x509.CertPool {
	t.Helper()
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(certPEM) {
		t.Fatal("failed to build expected certificate pool")
	}
	return pool
}
