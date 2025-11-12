package persistence

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

// SchemaValidator validates payloads against JSON Schemas compiled via santhosh-tekuri/jsonschema.
type SchemaValidator struct {
	mu    sync.RWMutex
	cache map[string]*jsonschema.Schema
}

// NewSchemaValidator returns a validator with an empty schema cache.
func NewSchemaValidator() *SchemaValidator {
	return &SchemaValidator{
		cache: make(map[string]*jsonschema.Schema),
	}
}

// Validate ensures the payload matches the provided schema definition.
func (v *SchemaValidator) Validate(ctx context.Context, schema SchemaRecord, payload []byte) error {
	if len(payload) == 0 {
		return fmt.Errorf("payload is required for validation")
	}

	compiled, err := v.getOrCompile(schema)
	if err != nil {
		return err
	}

	var document any
	if err := json.Unmarshal(payload, &document); err != nil {
		return fmt.Errorf("decode payload: %w", err)
	}

	if err := compiled.Validate(document); err != nil {
		return fmt.Errorf("schema validation: %w", err)
	}

	return nil
}

func (v *SchemaValidator) getOrCompile(schema SchemaRecord) (*jsonschema.Schema, error) {
	key := v.cacheKey(schema)

	v.mu.RLock()
	compiled, ok := v.cache[key]
	v.mu.RUnlock()
	if ok {
		return compiled, nil
	}

	v.mu.Lock()
	defer v.mu.Unlock()

	// another goroutine may have populated the cache while we were waiting
	if compiled, ok = v.cache[key]; ok {
		return compiled, nil
	}

	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource(key, bytes.NewReader(schema.SchemaDefinition)); err != nil {
		return nil, fmt.Errorf("register schema %s: %w", key, err)
	}

	newCompiled, err := compiler.Compile(key)
	if err != nil {
		return nil, fmt.Errorf("compile schema %s: %w", key, err)
	}

	v.cache[key] = newCompiled
	return newCompiled, nil
}

func (v *SchemaValidator) cacheKey(schema SchemaRecord) string {
	return fmt.Sprintf("memory://schemas/%s/%s", schema.SchemaID.String(), schema.VersionString())
}
