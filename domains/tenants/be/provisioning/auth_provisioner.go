package provisioning

import (
	"context"

	"github.com/zenGate-Global/palmyra-pro-saas/domains/tenants/be/service"
)

// AuthProvisioner is a placeholder that marks external auth tenant as ready.
// Replace with Firebase/Identity Platform ensure/check when available.
type AuthProvisioner struct{}

func NewAuthProvisioner() *AuthProvisioner { return &AuthProvisioner{} }

func (a *AuthProvisioner) Ensure(ctx context.Context, externalTenant string) (service.AuthProvisionResult, error) {
	if externalTenant == "" {
		return service.AuthProvisionResult{Ready: false}, nil
	}
	return service.AuthProvisionResult{Ready: true}, nil
}

func (a *AuthProvisioner) Check(ctx context.Context, externalTenant string) (service.AuthProvisionResult, error) {
	return a.Ensure(ctx, externalTenant)
}

var _ service.AuthProvisioner = (*AuthProvisioner)(nil)
