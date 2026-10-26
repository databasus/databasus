// Package httpapi serves the operations of the storages tag through the generated strict server,
// and converts between the settings of each provider and their contract schema.
package httpapi

//go:generate go tool oapi-codegen -config oapi-codegen.yaml ../../../../../api/openapi.yaml
