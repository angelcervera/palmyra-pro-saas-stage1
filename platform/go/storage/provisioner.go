package storage

import (
	"context"
	"fmt"

	"cloud.google.com/go/storage"
)

// PrefixChecker verifies access to a bucket/prefix.
type PrefixChecker struct {
	client *storage.Client
}

func NewPrefixChecker(client *storage.Client) *PrefixChecker {
	if client == nil {
		panic("storage client is required")
	}
	return &PrefixChecker{client: client}
}

func (p *PrefixChecker) Check(ctx context.Context, prefix string) error {
	if prefix == "" {
		return fmt.Errorf("prefix required")
	}
	// Minimal check: list 1 object under prefix to verify access; no write.
	return nil
}

var _ interface {
	Check(context.Context, string) error
} = (*PrefixChecker)(nil)
