## MODIFIED Requirements

### Requirement: A notification does not hold up the work that sends it

A backup, a WAL stream, a healthcheck or a verification that notifies the notifiers of a database SHALL continue without waiting for the deliveries. Every notifier SHALL get its own delivery, so a slow or failing notifier SHALL NOT delay or prevent the others. A delivery that has not finished after 30 seconds SHALL be stopped and SHALL count as failed. A delivery to Slack SHALL instead have 3 minutes, so it can wait out a rate limit, while each of its requests stays limited to 30 seconds. A Slack rate limit that asks for a longer wait than the delivery or the test send has left SHALL fail it at once. A test send SHALL keep the 30-second limit for every notifier type. A message longer than 2000 characters SHALL be cut to 2000 characters before it is sent.

#### Scenario: Webhook endpoint that never answers

- **WHEN** a database has a webhook notifier whose endpoint accepts the connection and never answers, and a Discord notifier, and its backup fails
- **THEN** the backup is recorded as failed without waiting for the webhook, the Discord notifier receives the failure message, and after 30 seconds the webhook notifier's last send error says the delivery timed out

#### Scenario: Long message

- **WHEN** a database's webhook notifier receives a notification whose message is 2500 characters long
- **THEN** the webhook endpoint receives exactly the first 2000 characters of the message

#### Scenario: Notifier that cannot be reached

- **WHEN** one of a database's two notifiers points at an address that refuses connections, and its healthcheck reports the database as unavailable
- **THEN** the other notifier receives the message

#### Scenario: Slack asks to wait thirty seconds

- **WHEN** Slack answers a delivery with a rate limit and `Retry-After: 30`, and accepts the next attempt
- **THEN** the message is delivered and the notifier's last send error stays empty

#### Scenario: Slack asks for a wait longer than the delivery has left

- **WHEN** Slack answers a delivery with a rate limit and `Retry-After: 3600`
- **THEN** the delivery fails at once and the notifier's last send error says Slack rate-limited the message

#### Scenario: Slack does not answer

- **WHEN** the Slack endpoint accepts a delivery's request and never answers
- **THEN** the request ends after 30 seconds, and the notifier's last send error is set and says the request timed out

#### Scenario: A Slack test send is rate-limited beyond its limit

- **WHEN** a member tests a saved Slack notifier and Slack answers with a rate limit and `Retry-After: 60`
- **THEN** the response is `200` and reports the message as not delivered without waiting out the wait, and the last send error says Slack rate-limited the message

#### Scenario: A Slack test send gets no answer

- **WHEN** a member tests a saved Slack notifier whose endpoint accepts the request and never answers
- **THEN** the test ends at the 30-second limit and reports the message as not delivered
