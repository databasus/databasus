# development-environment-check Specification

## Purpose

Tells a developer or an agent, before work starts, whether the machine they work on meets the prerequisites the project's commands and tests rely on, and how to fix each one that is missing.

## Requirements

### Requirement: The development environment check reports each prerequisite

The development environment check SHALL report every prerequisite it verifies on its own line, with the prerequisite's name and its outcome: passed or failed. A failed line SHALL say what was found and SHALL name the section of the VM setup guide that fixes it. The report SHALL list every check, even after an earlier one fails.

#### Scenario: One prerequisite is missing

- **WHEN** a developer runs the development environment check on a machine where only the bridge netfilter module is missing
- **THEN** the report shows that check as failed with the section of the VM setup guide that loads the module
- **AND** every other check is still reported with its own outcome

### Requirement: A missing prerequisite fails the check

The check SHALL fail when any prerequisite it verifies is missing: the kernel settings of the CloudNativePG tests, Docker and Docker Compose, the client tool bundles, the pinned tools, the root filesystem, the machine's size, the system and the commit hook. No prerequisite SHALL only produce a warning.

#### Scenario: Every prerequisite is met

- **WHEN** the check runs on a machine that meets every prerequisite
- **THEN** the check passes

#### Scenario: A prerequisite is missing

- **WHEN** the check runs on a machine that misses one prerequisite
- **THEN** the check fails

### Requirement: The check verifies the kernel settings of the CloudNativePG tests

The check SHALL fail unless the kernel applies iptables rules to bridged traffic, that is unless the bridge netfilter setting for iptables exists and is on. It SHALL also fail when the inotify limit on instances per user is below 1024 or the limit on watches per user is below 1048576. Higher limits SHALL pass.

#### Scenario: Bridge netfilter is missing

- **WHEN** the bridge netfilter module is neither loaded nor built into the kernel
- **THEN** the bridge netfilter check fails

#### Scenario: Bridge netfilter is turned off

- **WHEN** the module is loaded but the kernel does not apply iptables rules to bridged traffic
- **THEN** the bridge netfilter check fails and says the setting is off

#### Scenario: inotify limits are the kernel defaults

- **WHEN** the machine keeps the kernel's default limit of 128 inotify instances per user
- **THEN** the inotify check fails and reports the current and the required value

#### Scenario: Limits are higher than required

- **WHEN** the machine allows 8192 inotify instances and 2097152 watches per user
- **THEN** the inotify check passes

### Requirement: The check verifies Docker and Docker Compose

The check SHALL fail when Docker does not answer, when the Docker daemon runs rootless or when the Docker Compose plugin is missing. A Docker command that does not answer within 10 seconds SHALL count as not answering, so a hung daemon cannot hang the check.

#### Scenario: Rootless Docker

- **WHEN** the Docker daemon the developer's user reaches runs rootless
- **THEN** the Docker check fails and says that the tests need rootful Docker

#### Scenario: Docker is not running

- **WHEN** no Docker daemon answers
- **THEN** the Docker check fails within 10 seconds and the other checks are still reported

#### Scenario: Compose plugin is missing

- **WHEN** Docker answers but the Docker Compose plugin is missing
- **THEN** the Docker Compose check fails and names the missing plugin

### Requirement: The check verifies that the client tool bundles find their libraries

The check SHALL fail when a client tool bundle of the machine's architecture holds a program that needs a shared library the system cannot provide. It SHALL look up libraries as the dynamic loader finds them in the caller's environment, without creating aliases for missing ones, and SHALL name each program and missing library.

#### Scenario: A system library is missing

- **WHEN** a program of a client tool bundle needs a shared library that is not installed
- **THEN** the client tool bundles check fails and names the program and the missing library

### Requirement: The check verifies that the pinned tools are installed

The check SHALL fail when a tool pinned in the repository's tool configuration, including a tool pinned for a single task, is not installed at its pinned version, and SHALL name each missing tool.

#### Scenario: A pinned tool is missing

- **WHEN** the pinned version of one tool is not installed
- **THEN** the pinned tools check fails, names that tool and points to the locked tool installation

### Requirement: The check verifies the root filesystem and the machine's size

The check SHALL fail when the root filesystem is not ext4, when the machine has fewer than 4 logical CPUs, or when the memory the kernel reports is below 30 GiB. These are minimums, not the reference machine, which is larger. The memory threshold stands for 32 GiB of memory: a machine with 32 GiB reports about 31.3 GiB after the kernel's own reservation, so the threshold sits 2 GiB lower.

#### Scenario: A smaller machine

- **WHEN** the machine has 2 logical CPUs
- **THEN** the CPUs check fails and says that the machine has fewer than the minimum of 4 logical CPUs

#### Scenario: A machine at the minimum

- **WHEN** the machine has an ext4 root, exactly 4 logical CPUs and 31.3 GiB of total memory
- **THEN** the ext4, CPUs and Memory checks pass

#### Scenario: Memory at the threshold

- **WHEN** the kernel reports exactly 30 GiB of total memory
- **THEN** the Memory check passes, and at 29.9 GiB it fails

#### Scenario: A guest with 16 GiB

- **WHEN** the machine reports 15.6 GiB of total memory
- **THEN** the Memory check fails and says that the machine has less than the minimum of 32 GiB of memory

#### Scenario: btrfs root

- **WHEN** the root filesystem is btrfs
- **THEN** the ext4 check fails and says that the tests need an ext4 root filesystem

### Requirement: The check verifies the system and the commit hook

The check SHALL fail when the system is not Ubuntu or when the repository's commit hook is not installed.

#### Scenario: Another distribution

- **WHEN** the system is Fedora
- **THEN** the Ubuntu check fails and says that the project's tooling targets Ubuntu

#### Scenario: No commit hook

- **WHEN** the repository's commit hook is not installed
- **THEN** the Commit hook check fails and points to the command that installs it

### Requirement: The check changes nothing

The check SHALL only read the machine's state. It SHALL NOT load kernel modules, change kernel settings, install packages or tools, start or stop containers, or ask for administrator rights, whatever the outcome.

#### Scenario: A failed check

- **WHEN** the check finds the bridge netfilter module missing
- **THEN** the module is still not loaded after the check and no password prompt appeared
