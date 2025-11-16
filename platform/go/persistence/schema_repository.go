package persistence

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// SchemaRepositoryTable defines the canonical table name that stores schema definitions.
const SchemaRepositoryTable = "schema_repository"

// SchemaRepositoryDDL contains the PostgreSQL definition for the schema repository table.
// It enforces semantic version strings, soft deletes, and a single active schema per id.
const SchemaRepositoryDDL = `
CREATE TABLE IF NOT EXISTS schema_repository (
    schema_id UUID NOT NULL,
    schema_version TEXT NOT NULL CHECK (schema_version ~ '^\d+\.\d+\.\d+$'),
    schema_definition JSONB NOT NULL,
    table_name TEXT NOT NULL CHECK (table_name ~ '^[a-z][a-z0-9_]*$'),
    slug TEXT NOT NULL CHECK (slug ~ '^[a-z0-9]+(?:-[a-z0-9]+)*$'),
    category_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    is_soft_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    is_active BOOLEAN NOT NULL DEFAULT FALSE,
    PRIMARY KEY (schema_id, schema_version),
    FOREIGN KEY (category_id) REFERENCES schema_categories(category_id)
);

CREATE UNIQUE INDEX IF NOT EXISTS schema_repository_active_schema_idx
    ON schema_repository(schema_id)
    WHERE is_active AND NOT is_soft_deleted;

CREATE UNIQUE INDEX IF NOT EXISTS schema_repository_table_name_idx
    ON schema_repository(table_name)
    WHERE NOT is_soft_deleted AND is_active;

CREATE UNIQUE INDEX IF NOT EXISTS schema_repository_slug_idx
    ON schema_repository(slug)
    WHERE NOT is_soft_deleted AND is_active;

CREATE INDEX IF NOT EXISTS schema_repository_category_idx
    ON schema_repository(category_id)
    WHERE NOT is_soft_deleted;
`

// SemanticVersion is a minimal semantic version representation (major.minor.patch).
type SemanticVersion struct {
	Major uint32
	Minor uint32
	Patch uint32
}

// ParseSemanticVersion builds a SemanticVersion from a string formatted as major.minor.patch.
func ParseSemanticVersion(input string) (SemanticVersion, error) {
	parts := strings.Split(input, ".")
	if len(parts) != 3 {
		return SemanticVersion{}, fmt.Errorf("invalid semantic version %q", input)
	}

	var version SemanticVersion
	for idx, part := range parts {
		value, err := strconv.ParseUint(part, 10, 32)
		if err != nil {
			return SemanticVersion{}, fmt.Errorf("invalid semantic version %q: %w", input, err)
		}

		switch idx {
		case 0:
			version.Major = uint32(value)
		case 1:
			version.Minor = uint32(value)
		case 2:
			version.Patch = uint32(value)
		}
	}

	return version, nil
}

// String renders the semantic version in major.minor.patch notation.
func (v SemanticVersion) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

// Compare returns -1, 0, or 1 depending on the lexical ordering of the versions.
func (v SemanticVersion) Compare(other SemanticVersion) int {
	if v.Major != other.Major {
		return compareUint32(v.Major, other.Major)
	}
	if v.Minor != other.Minor {
		return compareUint32(v.Minor, other.Minor)
	}
	if v.Patch != other.Patch {
		return compareUint32(v.Patch, other.Patch)
	}
	return 0
}

// NextPatch returns a copy of the version with the patch segment incremented by one.
func (v SemanticVersion) NextPatch() SemanticVersion {
	return SemanticVersion{
		Major: v.Major,
		Minor: v.Minor,
		Patch: v.Patch + 1,
	}
}

func compareUint32(a, b uint32) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}

// SchemaDefinition is a JSON payload that holds the canonical schema definition body.
type SchemaDefinition = json.RawMessage

// SchemaRecord maps 1:1 with the schema_repository table, capturing every stored schema document.
type SchemaRecord struct {
	SchemaID         uuid.UUID        `db:"schema_id" json:"schemaId"`
	SchemaVersion    SemanticVersion  `db:"schema_version" json:"schemaVersion"`
	SchemaDefinition SchemaDefinition `db:"schema_definition" json:"schemaDefinition"`
	TableName        string           `db:"table_name" json:"tableName"`
	Slug             string           `db:"slug" json:"slug"`
	CategoryID       uuid.UUID        `db:"category_id" json:"categoryId"`
	CreatedAt        time.Time        `db:"created_at" json:"createdAt"`
	IsSoftDeleted    bool             `db:"is_soft_deleted" json:"isSoftDeleted"`
	IsActive         bool             `db:"is_active" json:"isActive"`
}

// VersionString returns the dotted semantic version for convenient SQL bindings.
func (r SchemaRecord) VersionString() string {
	return r.SchemaVersion.String()
}
