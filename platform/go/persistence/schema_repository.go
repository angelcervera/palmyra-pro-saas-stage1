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
