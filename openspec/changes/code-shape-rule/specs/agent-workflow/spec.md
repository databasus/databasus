## ADDED Requirements

### Requirement: The quick code-shape check reviews how new code is split
When a finished turn changes a Go or TypeScript source file of the backend, the verification agent, the frontend or the website, the quick code-shape check SHALL review the code the turn added or changed on two counts. First, the split: each new file and each new function holds one concept, and a turn that adds a second, unrelated concept to an existing file moves it into a file of its own. Second, the names and comments that the rules of the repository govern. The check SHALL reject the turn for each broken rule, name the file or the unit and the rule, and suggest the split or the name that follows the rule. The check SHALL NOT judge where code belongs across packages, modules and layers, which the quick architecture check judges. It SHALL NOT reject an addition to a file whose content the architecture document or a module doc fixes, such as the public contract file of a module or the errors file of a package. It SHALL NOT reject a turn for concepts that were already mixed in a file before the turn.

#### Scenario: A new file holds two concepts
- **WHEN** a turn adds a source file that holds a token store and an unrelated mail template renderer
- **THEN** the quick code-shape check rejects the turn, names the file and the rule, and suggests a file for each concept

#### Scenario: A new function does two things
- **WHEN** a turn adds a function that validates a request and also sends the notification mail
- **THEN** the quick code-shape check rejects the turn, names the function and the rule, and suggests a function for each concept

#### Scenario: An event added to the public contract of a module
- **WHEN** a turn adds an event to the file that holds a module's whole public contract, as the architecture document requires
- **THEN** the quick code-shape check does not reject the turn for that file

#### Scenario: A new concept appended to an existing file
- **WHEN** a turn adds a rate limiter to an existing file that holds only a request ID middleware
- **THEN** the quick code-shape check rejects the turn and suggests moving the rate limiter into a file of its own

#### Scenario: An edit inside a file that was already mixed
- **WHEN** a turn changes one function in a file that already held two concepts before the turn, and adds no new concept to it
- **THEN** the quick code-shape check does not reject the turn for the mixed file

#### Scenario: A unit in the wrong layer
- **WHEN** a turn adds a well-split file with a clear name to a package of a layer that must not hold it
- **THEN** the quick code-shape check does not report the placement, and the quick architecture check judges it

#### Scenario: A comment that restates the code
- **WHEN** a turn adds a comment above a function that only says what the function does
- **THEN** the quick code-shape check rejects the turn and suggests a name for the function that makes the comment redundant
