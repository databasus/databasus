# Strict server template

Both API contracts use `strict-gin.tmpl` to pass the HTTP request's context to application handlers. Gin contexts remain inside transport adapters and middleware because Gin reuses them after a request finishes. The architectural rule is in [HTTP API](../../../docs/architecture.md#http-api).

The override is based on `oapi-codegen` v2.8.0's `pkg/codegen/templates/strict/strict-gin.tmpl`. Both strict-server configurations register it as `strict/strict-gin.tmpl`. Template paths resolve relative to the generator's package; generation and the generated-output check run from that directory.

When upgrading the pinned generator, compare the override with the new upstream template and retain only the request-context change. Update this version reference, regenerate, and run the context regressions for both contracts plus the generated-output check. From the repository root, with the [backend test prerequisites](../../../docs/testing.md#commands) prepared:

```sh
mise run generate-api
TESTCONTAINERS_RYUK_DISABLED=true go -C backend test -count=1 -run '^TestStrictRequestContext$' \
  ./internal/lib/httpserver ./internal/modules/verification
mise run backend:lint
```
