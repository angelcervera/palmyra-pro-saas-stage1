package provisioning

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/zenGate-Global/palmyra-pro-saas/domains/tenants/be/service"
)

// LocalStorageProvisioner checks/creates a local filesystem prefix under BasePath.
type LocalStorageProvisioner struct {
	BasePath string
}

func NewLocalStorageProvisioner(basePath string) *LocalStorageProvisioner {
	if basePath == "" {
		panic("local storage provisioner requires basePath")
	}
	return &LocalStorageProvisioner{BasePath: basePath}
}

func (p *LocalStorageProvisioner) Check(ctx context.Context, prefix string) (service.StorageProvisionResult, error) {
	if prefix == "" {
		return service.StorageProvisionResult{Ready: false}, fmt.Errorf("storage prefix is required")
	}
	fullPath := filepath.Join(p.BasePath, prefix)
	// Ensure directory exists; this is safe/idempotent for local dev.
	if err := os.MkdirAll(fullPath, 0o755); err != nil {
		return service.StorageProvisionResult{Ready: false}, fmt.Errorf("create prefix path: %w", err)
	}
	return service.StorageProvisionResult{Ready: true}, nil
}

var _ service.StorageProvisioner = (*LocalStorageProvisioner)(nil)
