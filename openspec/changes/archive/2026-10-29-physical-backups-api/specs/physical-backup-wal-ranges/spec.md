## MODIFIED Requirements

### Requirement: WAL range rows offer no actions

A WAL range row SHALL offer no delete, download, cancel or restore action, as WAL segment rows did not. The API SHALL NOT delete a WAL segment either: a deletion by the ID of a WAL segment SHALL get `404` with `code` `not_found` and SHALL remove nothing.

#### Scenario: Actions column for a range

- **WHEN** the list shows a WAL range row to a user who can manage the database
- **THEN** the row's actions cell is empty

#### Scenario: Deleting a WAL segment through the API

- **WHEN** a member of the database's workspace deletes a physical backup by the ID of an archived WAL segment
- **THEN** the response is `404` with `code` `not_found`
- **AND** the WAL range that holds the segment is listed with the same count and size as before
