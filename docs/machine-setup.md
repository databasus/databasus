# Preparing a machine

Local tests create and remove many containers and temporary files. Their speed depends on storage latency for small writes and filesystem synchronization, as well as disk throughput.

Keep temporary test files and Docker's writable container data on fast local storage. A memory-backed temporary directory can help when enough RAM is available. Moving only the host's temporary directory does not move Docker's own data; check both locations when investigating slow runs.

Keep valuable container data and persistent volumes on reliable storage. Disabling write-durability guarantees can cause data loss after a power failure or crash, and changing shared Docker storage affects every project on the machine.

These optimizations are optional and depend on the host. No repository command requires a particular filesystem or mount configuration. Tool installation follows the root [development guide](../AGENTS.md#development-environment); [testing.md](testing.md#checking-work) explains which checks to run.
