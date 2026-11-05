## ADDED Requirements

### Requirement: The verify action of a logical backup starts one check per confirmation

The logical backup list SHALL keep one verify confirmation open at a time, so repeated use of a row's verify action starts one restore check. Once that confirmation is declined or its request settles, the action SHALL work again.

#### Scenario: The verify action is used twice before the confirmation is answered

- **WHEN** the user activates a row's verify action twice and confirms what appears
- **THEN** one check request is sent

#### Scenario: A refused check can be tried again

- **WHEN** the server refuses a confirmed check and the user activates the row's verify action again
- **THEN** the translated reason appears, a new confirmation opens, and confirming it sends another check request

#### Scenario: A declined confirmation does not block the action

- **WHEN** the user declines the confirmation and activates the row's verify action again
- **THEN** a new confirmation opens
