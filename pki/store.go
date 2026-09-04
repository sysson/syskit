package pki

import (
	"context"
	"errors"
)

// ErrNoLeaf is returned by Store.LoadLeaf when name has no certificate.
var ErrNoLeaf = errors.New("no certificate found")

// Store persists and retrieves the CA and leaf keypairs an Authority issues,
// independent of backend (local disk, Kubernetes Secret, ...). A Store holds
// a single CA plus any number of named leaves, each optionally paired with
// the CA certificate (never its private key) it was issued by, so a leaf can
// be consumed on its own without a separate trip to fetch the CA.
type Store interface {
	// SaveCA persists the CA keypair, including its private key.
	SaveCA(ctx context.Context, ca KeyPair) error
	// LoadCA returns the previously saved CA keypair, or an error wrapping
	// ErrNoAuthority if none has been saved.
	LoadCA(ctx context.Context) (KeyPair, error)

	// SaveLeaf persists leaf under name, alongside ca's certificate (never
	// its private key) if ca is non-empty.
	SaveLeaf(ctx context.Context, name string, ca, leaf KeyPair) error
	// LoadLeaf returns the leaf saved under name and its companion CA
	// certificate if one was saved, or an error wrapping ErrNoLeaf if name
	// has no certificate.
	LoadLeaf(ctx context.Context, name string) (ca, leaf KeyPair, err error)
	// DeleteLeaf removes the leaf saved under name, if any.
	DeleteLeaf(ctx context.Context, name string) error
	// ListLeaves returns the names of every leaf saved in the store, sorted
	// alphabetically.
	ListLeaves(ctx context.Context) ([]string, error)
}
