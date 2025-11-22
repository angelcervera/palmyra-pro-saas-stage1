package provisioning

import (
	"context"
	"fmt"

	"github.com/zenGate-Global/palmyra-pro-saas/domains/tenants/be/service"
)

// AuthProvisioner is a placeholder; replace with real Firebase/Identity logic later.
// TODO: wire Firebase Admin SDK client and perform tenant ensure/check.
type AuthProvisioner struct{}

func NewAuthProvisioner() *AuthProvisioner { return &AuthProvisioner{} }

func (a *AuthProvisioner) Ensure(ctx context.Context, externalTenant string) (service.AuthProvisionResult, error) {
	return service.AuthProvisionResult{Ready: false}, fmt.Errorf("auth provisioner not implemented")
}

func (a *AuthProvisioner) Check(ctx context.Context, externalTenant string) (service.AuthProvisionResult, error) {
	return a.Ensure(ctx, externalTenant)
}

var _ service.AuthProvisioner = (*AuthProvisioner)(nil)
