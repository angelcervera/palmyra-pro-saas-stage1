package provisioning

import (
	"context"
	"fmt"

	"github.com/zenGate-Global/palmyra-pro-saas/domains/tenants/be/service"
)

// StorageProvisioner is a placeholder; replace with real GCS checks later.
// TODO: inject GCS client and verify prefix/bucket access.
type StorageProvisioner struct{}

func NewStorageProvisioner() *StorageProvisioner { return &StorageProvisioner{} }

func (s *StorageProvisioner) Check(ctx context.Context, prefix string) (service.StorageProvisionResult, error) {
	return service.StorageProvisionResult{Ready: false}, fmt.Errorf("storage provisioner not implemented")
}

var _ service.StorageProvisioner = (*StorageProvisioner)(nil)
