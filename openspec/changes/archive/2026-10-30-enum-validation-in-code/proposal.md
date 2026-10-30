## Why

The refactoring adds closed SQL value lists that reject settings accepted by the previous release. An existing S3-compatible storage class can therefore prevent startup; removing only the database constraint would still leave that setting unusable through the API.

## What Changes

- Apply the enum storage rule already recorded in `backend/AGENTS.md`: application validation owns allowed values; migrations store scalar values as TEXT and lists as TEXT[] without enum value CHECK constraints or PostgreSQL enum types.
- Edit only the unpublished refactoring migrations. Remove scalar, array-membership and fixed-discriminator enum checks, while preserving structural constraints and necessary upgrade transformations.
- Review both migration directions and dependent migration tests so no statement references a removed constraint.
- Accept and preserve an arbitrary S3-compatible storage class through migration, API, provider and editing UI, without replacing it with a default.
- Add an upgrade regression covering classes outside the current suggestions.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `internal-database-migrations`: upgrades preserve provider-defined storage classes without closing the database's text values over an application enum.
- `storages`: S3-compatible storage classes are provider-defined text that survives reads and edits.
- `backup-configs`: upgrade normalization is justified by application validation rather than by enum CHECK constraints.

## Impact

Constrained by root `AGENTS.md`, `backend/AGENTS.md`, `frontend/AGENTS.md`, the testing skill and the API-design skill. Affects unpublished SQL migrations and migration tests, the S3 settings/provider and contract, generated clients/types, the storage editor and upgrade fixtures. The instruction and its architecture link are recorded during planning; SQL and application changes remain implementation tasks.

Removing value constraints changes the unpublished schema. The S3 API change widens accepted values; HTTP paths and field names stay the same. No runtime dependencies are added.

### Out of scope

- Rewriting released migrations or supporting databases initialized from intermediate unpublished refactoring commits.
- Removing structural CHECK constraints, foreign keys or unique indexes.
- Weakening validation of actual application enums.
- Repairing the webhook filter behavior in R2.
- Removing rclone providers or its accepted dependency exceptions.
