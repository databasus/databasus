// Package api holds the HTTP contracts that the servers, the test clients and the frontend types
// are generated from.
package api

import _ "embed"

//go:embed openapi.yaml
var OpenAPI []byte

//go:embed agent.yaml
var Agent []byte
