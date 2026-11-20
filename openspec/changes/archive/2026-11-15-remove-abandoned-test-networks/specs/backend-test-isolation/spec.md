## ADDED Requirements

### Requirement: Abandoned test networks are removed

Test networks are the Docker networks that have at least one subnet and whose every subnet lies in the address range reserved for networks that tests create. A test network is abandoned when it was created more than an hour ago and has no running container. Every backend test process that creates a network SHALL try to remove the abandoned test networks once, whether or not the earlier run was started through the project's test runner. A backend test process that runs until its removal attempt ends SHALL leave no abandoned test network that existed when the attempt began, apart from networks Docker refused to remove; networks still present when a process exits earlier SHALL be removed by later backend test processes in the same way. A test network with a running container SHALL NOT be removed, whatever its age. A network without a subnet, or with a subnet outside the reserved range, SHALL NOT be removed, whatever its age and labels. Browser end-to-end runs do not remove networks themselves; their abandoned networks are removed by backend test processes.

#### Scenario: A killed run left networks behind

- **WHEN** a direct test run was killed two hours ago and left empty test networks, and a new backend test package creates a network and runs until its removal attempt ends
- **THEN** those networks no longer exist

#### Scenario: Docker restarted during a run

- **WHEN** Docker restarted during a run more than an hour ago, so the containers of that run are stopped and their test networks have no running container, and a new backend test package creates a network and runs until its removal attempt ends
- **THEN** those networks no longer exist

#### Scenario: A neighbouring run's network is in use

- **WHEN** another run's test network was created more than an hour ago and a running container is attached to it
- **THEN** a new backend test process does not remove it

#### Scenario: A neighbouring run just created its network

- **WHEN** another run created an empty test network less than an hour ago
- **THEN** a new backend test process does not remove it

#### Scenario: Networks of other systems are left alone

- **WHEN** an empty network outside the reserved range, such as a verification agent's restore network or another project's network, was created more than an hour ago
- **THEN** no backend test process removes it

#### Scenario: A network without a subnet is left alone

- **WHEN** a network has no subnet at all, such as Docker's `host` or `none` network or another project's network without address management
- **THEN** no backend test process removes it
