## REMOVED Requirements

### Requirement: A separate browser suite exercises the production frontend against a real backend

**Reason**: The suite no longer belongs to the frontend. It runs from the root `e2e/` directory and imports nothing from the frontend or the backend.

**Migration**: Run `mise run e2e:browser`. The `browser-e2e` capability states what the suite proves. The rules that constrained the deleted test gateway are dropped, because the application now serves itself: runtime configuration before the bundle, one origin for frontend and backend, and account setup through the API, which registration through the interface replaces. `pnpm --dir frontend test:browser --project=ui` is unchanged and needs no backend.

### Requirement: Browser end-to-end failures propagate through every lifecycle owner

**Reason**: The lifecycle owners this requirement names are deleted with the old suite.

**Migration**: The `browser-e2e` capability covers this ground in its requirements on cleanup, crashed runs, overlapping runs, secrets in a run's output and lifecycle tests. The rule on authorization codes is dropped, because no scenario of the new suite goes through an authorization flow.
