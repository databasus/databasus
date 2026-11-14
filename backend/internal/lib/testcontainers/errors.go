package containers

import "errors"

// Each names a host prerequisite of the disposable Kubernetes clusters, so a missing one fails the
// suite with its name instead of a timeout.
var (
	ErrDockerUnavailable           = errors.New("docker is unavailable")
	ErrPrivilegedContainerRequired = errors.New("k3s requires a privileged container")
	ErrContainerAddressUnreachable = errors.New(
		"rootful Docker on Linux is required: the k3s container address is unreachable from the host",
	)
	ErrImageUnavailable        = errors.New("a pinned image cannot be obtained")
	ErrBridgeNetfilterRequired = errors.New(
		"the br_netfilter kernel module is required: load it on the host with modprobe br_netfilter",
	)
	ErrKubernetesCommandFailed = errors.New("kubernetes command failed")
	ErrCNPGManifestChanged     = errors.New("the pinned CloudNativePG manifest does not have the expected content")
	ErrK3sStartTimedOut        = errors.New("k3s did not start; the host is likely short of CPU")
)
