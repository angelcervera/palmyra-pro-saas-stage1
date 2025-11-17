-- Development seed data for Palmyra Pro.
-- These inserts are idempotent to keep `docker compose up` repeatable.

INSERT INTO schema_categories (category_id, name, slug, description, created_at, updated_at)
VALUES
    ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa1', 'Digital Collectibles', 'digital-collectibles', 'Schemas for NFT-style items', NOW(), NOW()),
    ('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbb2', 'Supply Chain', 'supply-chain', 'Schemas for logistics docs', NOW(), NOW())
ON CONFLICT (category_id) DO NOTHING;

-- Activate a base schema for collectibles.
INSERT INTO schema_repository (
    schema_id,
    schema_version,
    schema_definition,
    table_name,
    slug,
    category_id,
    created_at,
    is_soft_deleted,
    is_active
) VALUES (
    'cccccccc-cccc-cccc-cccc-ccccccccccc3',
    '1.0.0',
    '{
      "$schema": "https://json-schema.org/draft/2020-12/schema",
      "type": "object",
      "properties": {
        "name": {"type": "string"},
        "rarity": {"type": "string"},
        "series": {"type": "string"}
      },
      "required": ["name"],
      "additionalProperties": false
    }'::jsonb,
    'digital_collectible_entities',
    'digital-collectible-schema',
    'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa1',
    NOW(),
    FALSE,
    TRUE
)
ON CONFLICT (schema_id, schema_version) DO UPDATE
SET schema_definition = EXCLUDED.schema_definition,
    table_name = EXCLUDED.table_name,
    slug = EXCLUDED.slug,
    category_id = EXCLUDED.category_id,
    is_active = TRUE,
    is_soft_deleted = FALSE;

-- Sample admin users for local testing.
INSERT INTO users (user_id, email, full_name, created_at, updated_at)
VALUES
    ('dddddddd-dddd-dddd-dddd-ddddddddddd4', 'admin@palmyra.dev', 'Palmyra Admin', NOW(), NOW()),
    ('eeeeeeee-eeee-eeee-eeee-eeeeeeeeeee5', 'manager@palmyra.dev', 'Schema Manager', NOW(), NOW())
ON CONFLICT (user_id) DO NOTHING;
