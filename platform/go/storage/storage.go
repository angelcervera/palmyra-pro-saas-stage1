package storage

import (
	"fmt"
	"strings"

	"github.com/zenGate-Global/palmyra-pro-saas/platform/go/tenant"
)

// ObjectLocation describes where a blob should live.
type ObjectLocation struct {
	Bucket   string
	FullPath string
}

// ResolveObjectLocation combines tenant base prefix and logical key into a bucket/path pair.
//   - bucket must come from deployment configuration (one bucket per environment class).
//   - tenant.Space.BasePrefix already includes envKey and trailing slash (e.g. "dev/acme-12345678/").
//   - logicalKey is a tenant-relative key such as
//     "entities/<table_name>/<entity_uuid>/<file_slug>/<semantic_version>/file.png".
func ResolveObjectLocation(space tenant.Space, bucket string, logicalKey string) (ObjectLocation, error) {
	bucket = strings.TrimSpace(bucket)
	if bucket == "" {
		return ObjectLocation{}, fmt.Errorf("bucket is required")
	}
	key := strings.TrimSpace(logicalKey)
	key = strings.TrimPrefix(key, "/")
	if key == "" {
		return ObjectLocation{}, fmt.Errorf("logical key is required")
	}

	prefix := space.BasePrefix
	if prefix == "" {
		return ObjectLocation{}, fmt.Errorf("tenant base prefix is missing")
	}

	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	fullPath := prefix + key
	return ObjectLocation{Bucket: bucket, FullPath: fullPath}, nil
}
