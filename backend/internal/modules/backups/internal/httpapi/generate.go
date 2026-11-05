// Package httpapi serves the operations of the backups tag through the generated strict server, and
// converts between the backup configs of each method and their contract schema.
package httpapi

//go:generate go tool oapi-codegen -config oapi-codegen.yaml ../../../../../api/openapi.yaml
