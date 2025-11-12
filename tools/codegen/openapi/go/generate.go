// This file triggers Go code generation from OpenAPI contracts.
// Run manually with:
//   go generate ./tools/codegen/openapi/go
//
// Generates code into /generated/go/<domain>/ following config files
// stored under /tools/codegen/openapi/go/configs/.

package main

// optional: shared components, only if you keep them separated

//go:generate go tool oapi-codegen -config ./configs/problemdetails.yaml ../../../../contracts/common/problemdetails.yaml
//go:generate go tool oapi-codegen -config ./configs/info.yaml ../../../../contracts/common/info.yaml
//go:generate go tool oapi-codegen -config ./configs/pagination.yaml ../../../../contracts/common/pagination.yaml
//go:generate go tool oapi-codegen -config ./configs/primitives.yaml ../../../../contracts/common/primitives.yaml
//go:generate go tool oapi-codegen -config ./configs/security.yaml ../../../../contracts/common/security.yaml
//go:generate go tool oapi-codegen -config ./configs/iam.yaml ../../../../contracts/common/iam.yaml

//go:generate go tool oapi-codegen -config ./configs/auth.yaml              ../../../../contracts/auth.yaml
//go:generate go tool oapi-codegen -config ./configs/users.yaml             ../../../../contracts/users.yaml
//go:generate go tool oapi-codegen -config ./configs/schema-categories.yaml ../../../../contracts/schema-categories.yaml
//go:generate go tool oapi-codegen -config ./configs/schema-repository.yaml ../../../../contracts/schema-repository.yaml
//go:generate go tool oapi-codegen -config ./configs/entities.yaml           ../../../../contracts/entities.yaml

func main() {}
