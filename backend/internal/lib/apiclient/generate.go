// Package apiclient is the client backend tests call the API through. It fails the test on every
// response that differs from the contract.
package apiclient

//go:generate go tool oapi-codegen -config oapi-codegen.yaml ../../../api/openapi.yaml
