// Package httpapi serves the operations of the users tag: the generated strict server, its
// handler and the middleware that enforces the access mark of every contract operation.
package httpapi

//go:generate go tool oapi-codegen -config oapi-codegen.yaml ../../../../../api/openapi.yaml
