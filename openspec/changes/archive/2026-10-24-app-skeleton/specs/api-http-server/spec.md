## ADDED Requirements

### Requirement: The application serves no API documentation interface

The application SHALL NOT serve an API documentation interface (Swagger UI) or the API description that interface was built from, in any mode: neither with `ENV_MODE=production` nor with `ENV_MODE=development`. Requests to the former paths of the interface and of the description SHALL get the same `404` response with a JSON body as any unknown path under `/api/`. The description was built from code comments the compiler does not check, and it described neither errors nor authentication. A published map of every operation also makes weak spots easier to find and is of no use to the user of an installed instance.

#### Scenario: Documentation requested on an installed instance

- **WHEN** a client requests `GET /api/v1/docs/swagger/index.html` from the application with `ENV_MODE=production`
- **THEN** the response status is `404` and the body is JSON with an error message

#### Scenario: API description requested on an installed instance

- **WHEN** a client requests `GET /api/v1/docs/swagger/doc.json` from the application with `ENV_MODE=production`
- **THEN** the response status is `404` and the body is JSON with an error message

#### Scenario: Documentation requested during local development

- **WHEN** a developer opens `/api/v1/docs/swagger/index.html` on the application with `ENV_MODE=development`
- **THEN** the response status is `404` and the body is JSON with an error message
