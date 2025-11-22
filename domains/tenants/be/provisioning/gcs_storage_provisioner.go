package provisioning

import (
	"context"
	"fmt"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"

	"github.com/zenGate-Global/palmyra-pro-saas/domains/tenants/be/service"
)

// GCSStorageProvisioner checks access to a GCS bucket/prefix.
type GCSStorageProvisioner struct {
	Client *storage.Client
	Bucket string
}

func NewGCSStorageProvisioner(client *storage.Client, bucket string) *GCSStorageProvisioner {
	if client == nil {
		panic("gcs storage provisioner requires client")
	}
	if bucket == "" {
		panic("gcs storage provisioner requires bucket")
	}
	return &GCSStorageProvisioner{Client: client, Bucket: bucket}
}

func (p *GCSStorageProvisioner) Check(ctx context.Context, prefix string) (service.StorageProvisionResult, error) {
	if prefix == "" {
		return service.StorageProvisionResult{Ready: false}, fmt.Errorf("storage prefix is required")
	}

	bkt := p.Client.Bucket(p.Bucket)
	if _, err := bkt.Attrs(ctx); err != nil {
		return service.StorageProvisionResult{Ready: false}, fmt.Errorf("bucket attrs: %w", err)
	}

	// List at most one object to validate access to the prefix; empty is fine.
	it := bkt.Objects(ctx, &storage.Query{Prefix: prefix})
	_, err := it.Next()
	if err != nil && err != iterator.Done {
		return service.StorageProvisionResult{Ready: false}, fmt.Errorf("list prefix: %w", err)
	}

	return service.StorageProvisionResult{Ready: true}, nil
}

var _ service.StorageProvisioner = (*GCSStorageProvisioner)(nil)
