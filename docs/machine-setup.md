# Preparing a machine

Local tests create and remove many containers and temporary files. Their speed depends on storage latency for small writes and filesystem synchronization, as well as disk throughput.

Keep temporary test files and Docker's writable container data on fast local storage. A memory-backed temporary directory can help when enough RAM is available. Moving only the host's temporary directory does not move Docker's own data; check both locations when investigating slow runs.

Keep valuable container data and persistent volumes on reliable storage. Disabling write-durability guarantees can cause data loss after a power failure or crash, and changing shared Docker storage affects every project on the machine.

These storage optimizations are optional and depend on the host. No repository command requires a particular filesystem or mount configuration. Tool installation follows the root [development guide](../AGENTS.md#development-environment); [testing.md](testing.md#checking-work) explains which checks to run.

## inotify limits for CloudNativePG tests

The [CloudNativePG suite](testing.md#cloudnativepg-physical-backup-tests) runs Kubernetes as k3s in Docker containers. The kubelet and containerd of every k3s container take inotify instances and watches from the per-user limits of the host, and the kernel default of 128 instances may not cover several k3s containers at once. CI raises both limits before the suite runs; set the same values on the host before running it locally:

```sh
sudo sysctl -w fs.inotify.max_user_instances=1024
sudo sysctl -w fs.inotify.max_user_watches=1048576
```

The values above last until the next reboot. To keep them, write them to a file under `/etc/sysctl.d` and reload:

```sh
sudo tee /etc/sysctl.d/90-databasus-k3s.conf >/dev/null <<'EOF'
fs.inotify.max_user_instances = 1024
fs.inotify.max_user_watches = 1048576
EOF
sudo sysctl --system
```

Higher values already set on the host can stay.

## Bridge netfilter for CloudNativePG tests

Kubernetes routes a pod's request to a service address through iptables rules. Traffic between pods of one k3s container crosses a Linux bridge, and the kernel applies those rules to bridged traffic only while the `br_netfilter` module is loaded. Without it, cluster DNS times out and CloudNativePG never adds the replica. k3s cannot load the module from inside its container, and a fixture fails at startup when the module is missing. Load it on the host:

```sh
sudo modprobe br_netfilter
```

The module stays loaded until the next reboot. To load it at every boot:

```sh
echo br_netfilter | sudo tee /etc/modules-load.d/br_netfilter.conf
```
