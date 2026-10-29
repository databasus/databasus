package container

// HostCapabilities is what the Docker daemon reports it can enforce, read once
// at startup because the kernel under the daemon does not change at runtime.
type HostCapabilities struct {
	IsUserNSRemapEnabled bool
	IsCPULimitSupported  bool
}
