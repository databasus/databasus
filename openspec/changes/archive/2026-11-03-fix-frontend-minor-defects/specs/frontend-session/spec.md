## MODIFIED Requirements

### Requirement: OAuth sign-in callbacks belong to one browser-bound attempt

Starting GitHub or Google sign-in SHALL create an unpredictable state bound to that provider, the initiating tab, the redirect address and a ten-minute lifetime. Only a callback carrying the matching state and an authorization code within that lifetime SHALL start code exchange with the recorded provider and redirect address. The attempt SHALL be consumed before exchange and SHALL permit at most one exchange. Missing, malformed, mismatched, replayed or expired attempts SHALL fail before exchange and offer a fresh sign-in. Provider refusal SHALL produce translated feedback without exchange. Every failure the callback finds before exchange SHALL use one translated message saying that sign-in did not finish and that the user should start again, and SHALL NOT attribute the failure to an invalid provider. Attempt metadata SHALL contain no password, client secret, access token or authorization code. Failure to create secure attempt metadata SHALL prevent navigation to the provider.

#### Scenario: A provider returns a matching callback

- **WHEN** either supported provider returns a code and matching state for a live attempt in its initiating tab
- **THEN** exactly one exchange uses that attempt's provider and redirect address and successful completion activates the session
- **AND** the attempt cannot be reused

#### Scenario: The callback is unrelated to a live attempt

- **WHEN** callback state is missing, incorrect, already consumed, expired or has no valid local attempt, including an old provider-name state, or the authorization code is missing
- **THEN** no code exchange occurs, the message says that sign-in did not finish and the user can restart sign-in

#### Scenario: Callback processing is started again

- **WHEN** callback processing repeats while its first exchange is pending
- **THEN** no second exchange starts and only the current callback lifecycle can accept the first completion

#### Scenario: A provider refuses authorization

- **WHEN** a provider returns a refusal for the current attempt
- **THEN** the attempt ends, no exchange occurs and the translated message says that sign-in did not finish and offers a fresh sign-in

#### Scenario: The server refuses the exchange

- **WHEN** a code exchange fails with a server refusal or a network failure
- **THEN** the translated reason for that failure is shown and the user can restart sign-in

#### Scenario: Attempt storage is unavailable

- **WHEN** secure state generation or tab-local attempt storage fails
- **THEN** the interface reports the failure and does not start an unbound authorization flow
