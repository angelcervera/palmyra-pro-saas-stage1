package provisioning

import (
	"context"

	"github.com/zenGate-Global/palmyra-pro-saas/domains/tenants/be/service"
)

// StorageProvisioner is a placeholder that validates prefix presence.
// Extend to perform real GCS prefix checks when GCS client wiring is added.
type StorageProvisioner struct{}

func NewStorageProvisioner() *StorageProvisioner { return &StorageProvisioner{} }

func (s *StorageProvisioner) Check(ctx context.Context, prefix string) (service.StorageProvisionResult, error) {
	if prefix == "" {
		return service.StorageProvisionResult{Ready: false}, nil
	}
	return service.StorageProvisionResult{Ready: true}, nil
}

var _ service.StorageProvisioner = (*StorageProvisioner)(nil)
