## ADDED Requirements

### Requirement: The verification agent keeps its token file private

The verification agent stores its token in its configuration file on the user's machine. The agent SHALL create that file readable and writable only by its owner, with mode `0600`, and SHALL keep that mode when it rewrites the file. At every start, before it reads the token, the agent SHALL set mode `0600` on an existing configuration file that has any permission bit for the group or for others, so that a file an earlier release created with mode `0644` is closed on the first start after the upgrade. A file with no group or other permission bit, such as one with mode `0400`, SHALL be left as it is.

When the agent cannot narrow the mode, for example because the file belongs to another user, it SHALL keep starting and verifying as before. It SHALL show a warning on the console of the command that starts it and SHALL write the same warning to its log. The warning SHALL be marked as a warning and SHALL name:

- the path of the configuration file;
- its current mode and its owner;
- the error the operating system returned;
- that other users of the machine can read the agent token;
- the command that sets mode `0600` on the file and, when the file belongs to another user, the command that gives it to the user the agent runs as.

The first line of the console warning SHALL start with `WARNING:`, and the log record SHALL carry a message that starts with `warning:`, because the agent's console and log lines show no level. The warning SHALL NOT contain the token. The agent SHALL show it at every start while the file still has a group or other permission bit.

#### Scenario: First save creates a private file

- **WHEN** the agent saves its configuration and no configuration file exists yet
- **THEN** the new file has mode `0600`

#### Scenario: Rewriting keeps the file private

- **WHEN** the agent rewrites a configuration file whose mode is `0600`
- **THEN** the file still has mode `0600`

#### Scenario: A file from an earlier release is narrowed

- **WHEN** the agent starts and its configuration file has mode `0644` and belongs to the user the agent runs as
- **THEN** the file has mode `0600` before the agent reads the token
- **AND** the agent starts without a warning about the file

#### Scenario: The mode cannot be narrowed

- **WHEN** the agent starts and its configuration file has mode `0644`, belongs to another user and the agent cannot change its mode
- **THEN** the agent starts and verifies backups as it would with a private file
- **AND** the console of the starting command shows a warning whose first line starts with `WARNING:`, which says that other users of the machine can read the agent token, and which names the file's path, its mode, its owner, the operating system's error, the command that sets mode `0600` and the command that changes the owner
- **AND** the agent's log holds a record whose message starts with `warning:` and that names the same path, mode, owner, error and commands
- **AND** neither the console nor the log contains the token

#### Scenario: The warning repeats until the file is fixed

- **WHEN** the previous start showed the warning because the mode could not be narrowed, and the agent starts again with the file unchanged
- **THEN** the agent shows the warning again on the console and in its log

#### Scenario: A file without group or other bits is left alone

- **WHEN** the agent starts and its configuration file has mode `0400`
- **THEN** the file keeps mode `0400` and the agent shows no warning about it
