package sqlassets

import _ "embed"

//go:embed schema/tenant_space/users.sql
var UsersSQL string

//go:embed schema/platform/entity_schemas.sql
var EntitySchemasSQL string

//go:embed schema/platform/tenants.sql
var TenantsSQL string
