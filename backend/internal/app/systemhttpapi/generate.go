// Package systemhttpapi serves the operations of the system tag. The system tag is not a module:
// its operations describe the running instance, which internal/app assembles.
package systemhttpapi

//go:generate go tool oapi-codegen -config oapi-codegen.yaml ../../../api/openapi.yaml
