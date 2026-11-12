## ADDED Requirements

### Requirement: The upgrade test does not collide with concurrent runs

The upgrade test SHALL be able to run beside other test runs on one machine, from one worktree or several. The network the test creates itself SHALL have a subnet inside the test network range that the testing documentation names, outside Docker's default address pools. The restore networks that the released and current verification agents create during the test still come from Docker's default pools, because an agent cannot choose its range. When the upgrade command builds the current image itself because no image is named, it SHALL tag that image with a name unique to the run, SHALL print that name, and SHALL remove the image when the test ends, whether it passes, fails or is interrupted. It SHALL NOT replace or remove an image that it did not build. An image named in `DATABASUS_IMAGE` SHALL be used as given and SHALL NOT be removed by the test.

#### Scenario: Two upgrade tests overlap

- **WHEN** two worktrees run the upgrade command without a named image at the same time
- **THEN** each command's output names a different image tag
- **AND** neither tag exists after both tests end

#### Scenario: The test fails or is interrupted

- **WHEN** an upgrade test that built its own image fails, or the developer interrupts it
- **THEN** no image with that run's tag remains afterwards

#### Scenario: A named image

- **WHEN** the upgrade command runs with `DATABASUS_IMAGE` naming an image
- **THEN** it builds no image, tests the named one and leaves it in place after the test ends

#### Scenario: The test's own network

- **WHEN** the upgrade test creates its network
- **THEN** that network's subnet lies inside the test network range that the testing documentation names
