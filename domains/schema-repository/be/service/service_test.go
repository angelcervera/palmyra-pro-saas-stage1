package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	domainrepo "github.com/zenGate-Global/palmyra-pro-saas/domains/schema-repository/be/repo"
	"github.com/zenGate-Global/palmyra-pro-saas/platform/go/persistence"
)

func TestServiceCreateSuccess(t *testing.T) {
	t.Parallel()

	repo := newFakeRepository()
	svc := New(repo)

	categoryID := uuid.New()

	created, err := svc.Create(context.Background(), CreateInput{
		Definition: json.RawMessage(`{"title":"schema-v1"}`),
		TableName:  "cards_entities",
		Slug:       "Cards-Schema",
		CategoryID: categoryID,
	})
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, created.SchemaID)
	require.Equal(t, "cards-schema", created.Slug)
	require.Equal(t, persistence.SemanticVersion{Major: 1, Minor: 0, Patch: 0}, created.Version)
	require.True(t, created.IsActive)
}

func TestServiceCreateConflict(t *testing.T) {
	t.Parallel()

	repo := newFakeRepository()
	svc := New(repo)

	initial, err := svc.Create(context.Background(), CreateInput{
		Definition: json.RawMessage(`{"title":"schema-v1"}`),
		TableName:  "cards_entities",
		Slug:       "cards-schema",
		CategoryID: uuid.New(),
	})
	require.NoError(t, err)

	_, err = svc.Create(context.Background(), CreateInput{
		SchemaID:   uuidPtr(initial.SchemaID),
		Version:    versionPtr(initial.Version),
		Definition: json.RawMessage(`{"title":"schema-v1-duplicate"}`),
		TableName:  "cards_entities",
		Slug:       "cards-schema",
		CategoryID: uuid.New(),
	})
	require.ErrorIs(t, err, ErrConflict)
}

func TestServiceCreateRejectsSlugChange(t *testing.T) {
	t.Parallel()

	repo := newFakeRepository()
	svc := New(repo)

	result, err := svc.Create(context.Background(), CreateInput{
		Definition: json.RawMessage(`{"title":"schema-v1"}`),
		TableName:  "cards_entities",
		Slug:       "cards-schema",
		CategoryID: uuid.New(),
	})
	require.NoError(t, err)

	_, err = svc.Create(context.Background(), CreateInput{
		SchemaID:   uuidPtr(result.SchemaID),
		Definition: json.RawMessage(`{"title":"schema-v2"}`),
		TableName:  "cards_entities",
		Slug:       "different-slug",
		CategoryID: uuid.New(),
	})

	var validationErr *ValidationError
	require.ErrorAs(t, err, &validationErr)
	require.Contains(t, validationErr.Fields, "slug")
}

func TestServiceListFiltersDeleted(t *testing.T) {
	t.Parallel()

	repo := newFakeRepository()
	svc := New(repo)

	first, err := svc.Create(context.Background(), CreateInput{
		Definition: json.RawMessage(`{"title":"schema-v1"}`),
		TableName:  "cards_entities",
		Slug:       "cards-schema",
		CategoryID: uuid.New(),
	})
	require.NoError(t, err)

	require.NoError(t, svc.Delete(context.Background(), first.SchemaID, first.Version))

	second, err := svc.Create(context.Background(), CreateInput{
		SchemaID:   uuidPtr(first.SchemaID),
		Definition: json.RawMessage(`{"title":"schema-v2"}`),
		TableName:  "cards_entities",
		Slug:       "cards-schema",
		CategoryID: uuid.New(),
	})
	require.NoError(t, err)

	list, err := svc.List(context.Background(), first.SchemaID, false)
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, second.Version, list[0].Version)

	listAll, err := svc.List(context.Background(), first.SchemaID, true)
	require.NoError(t, err)
	require.Len(t, listAll, 2)
}

func TestServiceListAll(t *testing.T) {
	t.Parallel()

	repo := newFakeRepository()
	svc := New(repo)

	first, err := svc.Create(context.Background(), CreateInput{
		Definition: json.RawMessage(`{"title":"schema-v1"}`),
		TableName:  "cards_entities",
		Slug:       "cards-schema",
		CategoryID: uuid.New(),
	})
	require.NoError(t, err)

	_, err = svc.Create(context.Background(), CreateInput{
		Definition: json.RawMessage(`{"title":"schema-two"}`),
		TableName:  "another_entities",
		Slug:       "schema-two",
		CategoryID: uuid.New(),
	})
	require.NoError(t, err)

	all, err := svc.ListAll(context.Background(), false)
	require.NoError(t, err)
	require.Len(t, all, 2)

	require.NoError(t, svc.Delete(context.Background(), first.SchemaID, first.Version))

	activeOnly, err := svc.ListAll(context.Background(), false)
	require.NoError(t, err)
	require.Len(t, activeOnly, 1)

	withDeleted, err := svc.ListAll(context.Background(), true)
	require.NoError(t, err)
	require.Len(t, withDeleted, 2)
}

func TestServiceActivateSwitchesActiveVersion(t *testing.T) {
	t.Parallel()

	repo := newFakeRepository()
	svc := New(repo)

	createdV1, err := svc.Create(context.Background(), CreateInput{
		Definition: json.RawMessage(`{"title":"schema-v1"}`),
		TableName:  "cards_entities",
		Slug:       "cards-schema",
		CategoryID: uuid.New(),
	})
	require.NoError(t, err)

	createdV2, err := svc.Create(context.Background(), CreateInput{
		SchemaID:   uuidPtr(createdV1.SchemaID),
		Definition: json.RawMessage(`{"title":"schema-v2"}`),
		TableName:  "cards_entities",
		Slug:       "cards-schema",
		CategoryID: uuid.New(),
	})
	require.NoError(t, err)

	activated, err := svc.Activate(context.Background(), createdV1.SchemaID, createdV2.Version)
	require.NoError(t, err)
	require.True(t, activated.IsActive)

	fetchedV1, err := svc.Get(context.Background(), createdV1.SchemaID, createdV1.Version)
	require.NoError(t, err)
	require.False(t, fetchedV1.IsActive)
}

func TestServiceDeleteNotFound(t *testing.T) {
	t.Parallel()

	repo := newFakeRepository()
	svc := New(repo)

	err := svc.Delete(context.Background(), uuid.New(), persistence.SemanticVersion{Major: 1, Minor: 0, Patch: 0})
	require.ErrorIs(t, err, ErrNotFound)
}

func extractTitle(t *testing.T, raw json.RawMessage) string {
	t.Helper()
	var payload map[string]string
	require.NoError(t, json.Unmarshal(raw, &payload))
	return payload["title"]
}

type fakeRepository struct {
	records map[uuid.UUID]map[string]persistence.SchemaRecord
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{
		records: make(map[uuid.UUID]map[string]persistence.SchemaRecord),
	}
}

func (f *fakeRepository) Upsert(ctx context.Context, params persistence.CreateSchemaParams) (persistence.SchemaRecord, error) {
	schemaMap, ok := f.records[params.SchemaID]
	if !ok {
		schemaMap = make(map[string]persistence.SchemaRecord)
		f.records[params.SchemaID] = schemaMap
	}

	versionKey := params.Version.String()

	record, exists := schemaMap[versionKey]
	now := time.Now().UTC()

	if exists {
		record.SchemaDefinition = cloneRaw(params.Definition)
		record.CategoryID = params.CategoryID
		record.Slug = params.Slug
		record.TableName = params.TableName
		record.IsSoftDeleted = false
		if params.Activate {
			f.deactivateAll(params.SchemaID)
		}
		record.IsActive = params.Activate
		schemaMap[versionKey] = record
		return record, nil
	}

	if params.Activate {
		f.deactivateAll(params.SchemaID)
	}

	record = persistence.SchemaRecord{
		SchemaID:         params.SchemaID,
		SchemaVersion:    params.Version,
		SchemaDefinition: cloneRaw(params.Definition),
		TableName:        params.TableName,
		Slug:             params.Slug,
		CategoryID:       params.CategoryID,
		CreatedAt:        now,
		IsActive:         params.Activate,
		IsSoftDeleted:    false,
	}

	schemaMap[versionKey] = record
	return record, nil
}

func (f *fakeRepository) GetByVersion(ctx context.Context, schemaID uuid.UUID, version persistence.SemanticVersion) (persistence.SchemaRecord, error) {
	schemaMap, ok := f.records[schemaID]
	if !ok {
		return persistence.SchemaRecord{}, persistence.ErrSchemaNotFound
	}

	record, ok := schemaMap[version.String()]
	if !ok || record.IsSoftDeleted {
		return persistence.SchemaRecord{}, persistence.ErrSchemaNotFound
	}

	return record, nil
}

func (f *fakeRepository) GetActive(ctx context.Context, schemaID uuid.UUID) (persistence.SchemaRecord, error) {
	schemaMap, ok := f.records[schemaID]
	if !ok {
		return persistence.SchemaRecord{}, persistence.ErrSchemaNotFound
	}

	for _, record := range schemaMap {
		if record.IsActive && !record.IsSoftDeleted {
			return record, nil
		}
	}

	return persistence.SchemaRecord{}, persistence.ErrSchemaNotFound
}

func (f *fakeRepository) List(ctx context.Context, schemaID uuid.UUID) ([]persistence.SchemaRecord, error) {
	schemaMap, ok := f.records[schemaID]
	if !ok {
		return nil, nil
	}

	results := make([]persistence.SchemaRecord, 0, len(schemaMap))
	for _, record := range schemaMap {
		results = append(results, record)
	}

	return results, nil
}

func (f *fakeRepository) ListAll(ctx context.Context, includeInactive bool) ([]persistence.SchemaRecord, error) {
	var results []persistence.SchemaRecord
	for _, schemaMap := range f.records {
		for _, record := range schemaMap {
			if record.IsSoftDeleted && !includeInactive {
				continue
			}
			if !includeInactive && !record.IsActive {
				continue
			}
			results = append(results, record)
		}
	}
	return results, nil
}

func (f *fakeRepository) GetLatestBySlug(ctx context.Context, slug string) (persistence.SchemaRecord, error) {
	var latest *persistence.SchemaRecord
	for _, schemaMap := range f.records {
		for _, record := range schemaMap {
			if record.Slug != slug || record.IsSoftDeleted {
				continue
			}
			if latest == nil || record.CreatedAt.After(latest.CreatedAt) {
				tmp := record
				latest = &tmp
			}
		}
	}
	if latest == nil {
		return persistence.SchemaRecord{}, persistence.ErrSchemaNotFound
	}
	return *latest, nil
}

func (f *fakeRepository) Activate(ctx context.Context, schemaID uuid.UUID, version persistence.SemanticVersion) error {
	schemaMap, ok := f.records[schemaID]
	if !ok {
		return persistence.ErrSchemaNotFound
	}

	record, ok := schemaMap[version.String()]
	if !ok || record.IsSoftDeleted {
		return persistence.ErrSchemaNotFound
	}

	f.deactivateAll(schemaID)

	record.IsActive = true
	schemaMap[version.String()] = record

	return nil
}

func (f *fakeRepository) SoftDelete(ctx context.Context, schemaID uuid.UUID, version persistence.SemanticVersion, deletedAt time.Time) error {
	schemaMap, ok := f.records[schemaID]
	if !ok {
		return persistence.ErrSchemaNotFound
	}

	record, ok := schemaMap[version.String()]
	if !ok || record.IsSoftDeleted {
		return persistence.ErrSchemaNotFound
	}

	record.IsActive = false
	record.IsSoftDeleted = true
	schemaMap[version.String()] = record
	return nil
}

func (f *fakeRepository) deactivateAll(schemaID uuid.UUID) {
	schemaMap := f.records[schemaID]
	for key, record := range schemaMap {
		record.IsActive = false
		schemaMap[key] = record
	}
}

func cloneRaw(raw json.RawMessage) json.RawMessage {
	if raw == nil {
		return nil
	}
	buf := make([]byte, len(raw))
	copy(buf, raw)
	return buf
}

var _ domainrepo.Repository = (*fakeRepository)(nil)

func versionPtr(v persistence.SemanticVersion) *persistence.SemanticVersion {
	return &v
}
