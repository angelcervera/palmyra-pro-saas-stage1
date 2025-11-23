package persistence

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"

	"github.com/zenGate-Global/palmyra-pro-saas/platform/go/tenant"
)

// fakeTx satisfies pgx.Tx and records Exec statements invoked.
type execCall struct {
	sql  string
	args []any
}

type fakeTx struct{ calls []execCall }

func (f *fakeTx) Begin(ctx context.Context) (pgx.Tx, error) {
	return nil, errors.New("not implemented")
}
func (f *fakeTx) Commit(ctx context.Context) error   { return nil }
func (f *fakeTx) Rollback(ctx context.Context) error { return nil }
func (f *fakeTx) CopyFrom(context.Context, pgx.Identifier, []string, pgx.CopyFromSource) (int64, error) {
	return 0, errors.New("not implemented")
}
func (f *fakeTx) SendBatch(context.Context, *pgx.Batch) pgx.BatchResults { return nil }
func (f *fakeTx) LargeObjects() pgx.LargeObjects                         { return pgx.LargeObjects{} }
func (f *fakeTx) Prepare(context.Context, string, string) (*pgconn.StatementDescription, error) {
	return &pgconn.StatementDescription{}, errors.New("not implemented")
}
func (f *fakeTx) Query(context.Context, string, ...any) (pgx.Rows, error) {
	return nil, errors.New("not implemented")
}
func (f *fakeTx) QueryRow(context.Context, string, ...any) pgx.Row { return nil }
func (f *fakeTx) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	f.calls = append(f.calls, execCall{sql: sql, args: args})
	return pgconn.CommandTag{}, nil
}
func (f *fakeTx) Conn() *pgx.Conn { return nil }

// fakePool returns a preconstructed transaction.
type fakePool struct{ tx *fakeTx }

func (p *fakePool) BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error) {
	return p.tx, nil
}

func TestTenantDBWithAdminSetsOnlySearchPath(t *testing.T) {
	ftx := &fakeTx{}
	db := &TenantDB{pool: &fakePool{tx: ftx}, adminSchema: "admin"}

	err := db.WithAdmin(context.Background(), func(tx pgx.Tx) error { return nil })
	require.NoError(t, err)
	require.Len(t, ftx.calls, 1)
	require.Contains(t, strings.ToLower(ftx.calls[0].sql), "set_config('search_path'")
	require.Equal(t, "admin", ftx.calls[0].args[0])
}

func TestTenantDBWithTenantSetsRoleAndSearchPath(t *testing.T) {
	ftx := &fakeTx{}
	db := &TenantDB{pool: &fakePool{tx: ftx}, adminSchema: "admin"}
	space := tenant.Space{SchemaName: "tenant_acme", RoleName: "tenant_acme_role"}

	err := db.WithTenant(context.Background(), space, func(tx pgx.Tx) error { return nil })
	require.NoError(t, err)
	require.Len(t, ftx.calls, 2)
	require.Contains(t, ftx.calls[0].sql, "tenant_acme_role")
	require.Contains(t, strings.ToLower(ftx.calls[1].sql), "set_config('search_path'")
	require.Equal(t, "tenant_acme, admin", ftx.calls[1].args[0])
}

func TestTenantDBWithTenantMissingRole(t *testing.T) {
	db := &TenantDB{pool: &fakePool{tx: &fakeTx{}}, adminSchema: "admin"}
	err := db.WithTenant(context.Background(), tenant.Space{SchemaName: "tenant_acme"}, func(tx pgx.Tx) error { return nil })
	require.Error(t, err)
	require.Contains(t, err.Error(), "tenant role is required")
}
