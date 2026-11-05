## MODIFIED Requirements

### Requirement: An upgrade keeps every backup config

An upgrade SHALL keep every saved config with its values, its storage and its notification triggers. A database that has a saved config of each method SHALL keep the one of its own method, and the other SHALL be deleted. A stored application-owned value that the current application cannot use SHALL be replaced by the default of its method during the upgrade, so the instance starts with a usable config. Provider-defined storage classes SHALL be preserved as the storage requirements state.

#### Scenario: A database with both configs

- **WHEN** an instance with a MySQL database that has both a logical and a physical config row is upgraded
- **THEN** the database's config is its logical config with its saved values

#### Scenario: A value the new constraints reject

- **WHEN** an instance with a logical config whose retention period is an unknown value is upgraded
- **THEN** the config reads with the default retention period of the logical method and keeps every other saved value
