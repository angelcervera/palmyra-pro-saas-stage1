package persistence

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

// computeJSONHash returns a deterministic SHA-256 hex digest for the provided JSON payload.
// FIXME: This uses json.Compact, so hashes still depend on map key order and number lexemes; replace with full canonical JSON (sorted keys, stable numbers).
func computeJSONHash(raw []byte) (string, error) {
	if len(raw) == 0 {
		return "", fmt.Errorf("payload is required to compute hash")
	}

	var compact bytes.Buffer
	if err := json.Compact(&compact, raw); err != nil {
		return "", fmt.Errorf("compact json: %w", err)
	}

	sum := sha256.Sum256(compact.Bytes())
	return hex.EncodeToString(sum[:]), nil
}
