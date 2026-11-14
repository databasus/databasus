package containers

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	dockercontainer "github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/mount"
	dockernetwork "github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/network"
)

// LabelK3sFixture marks every container, network and volume of one k3s fixture with its ID. A killed
// direct test run leaves them behind with no run label, and this label is how they are found.
const LabelK3sFixture = "databasus.k3s-fixture"

const (
	k3sImage               = "rancher/k3s:v1.36.5-k3s1"
	k3sVersion             = "v1.36.5+k3s1"
	k3sAPIPort             = 6443
	k3sDataDirectory       = "/var/lib/rancher/k3s"
	k3sRegistriesFile      = "/etc/databasus-registries.yaml"
	k3sEntrypointFile      = "/etc/databasus-k3s-entrypoint.sh"
	k3sStartupLimit        = 3 * time.Minute
	k3sTotalStartLimit     = 5 * time.Minute
	k3sAddressDialLimit    = 5 * time.Second
	k3sDiagnosticsLimit    = 8 * 1024
	k3sPollInterval        = 500 * time.Millisecond
	k3sTerminateLimit      = 2 * time.Minute
	k3sServerAuthorityFile = "/var/lib/rancher/k3s/server/tls/server-ca.crt"
)

// k3sMaxPods sits well above the per-k3s capacity of CloudNativePG clusters, so the kubelet never
// refuses a pod before the pool starts another k3s. Each CNPG cluster runs two PostgreSQL pods and,
// during bootstrap and joins, short-lived job pods.
const k3sMaxPods = 250

// Every registry has a mirror without endpoints, and the default endpoint is disabled, so a missing
// imported image fails the pod instead of downloading an unpinned one.
const k3sRegistries = "mirrors:\n  \"*\": {}\n"

// k3sEntrypoint readies cgroup v2 before k3s starts. Inside the container's cgroup namespace the
// kernel refuses to enable controllers for child cgroups while a process sits in the root cgroup, and
// every docker exec the fixture runs lands there. k3s moves its processes out and enables the
// controllers once; an exec arriving at that moment leaves the kubelet without them and k3s exits.
// The entrypoint repeats both steps until the controllers the kubelet requires are enabled. Without
// a writable cgroup tree, an unprivileged container, it starts k3s at once so k3s reports why.
const k3sEntrypoint = `set -u
cgroup=/sys/fs/cgroup
if [ -w "$cgroup/cgroup.subtree_control" ]; then
    mkdir -p "$cgroup/init"
    read -r controllers <"$cgroup/cgroup.controllers"
    attempt=0
    while :; do
        xargs -rn1 <"$cgroup/cgroup.procs" >"$cgroup/init/cgroup.procs" 2>/dev/null || true
        for controller in $controllers; do
            echo "+$controller" >"$cgroup/cgroup.subtree_control" 2>/dev/null || true
        done
        missing=""
        for controller in cpuset cpu memory hugetlb pids; do
            case " $controllers " in *" $controller "*) ;; *) continue ;; esac
            case " $(cat "$cgroup/cgroup.subtree_control") " in *" $controller "*) ;; *) missing="$missing $controller" ;; esac
        done
        [ -z "$missing" ] && break
        attempt=$((attempt + 1))
        if [ "$attempt" -ge 100 ]; then
            echo "cannot enable cgroup controllers:$missing" >&2
            exit 1
        fi
        sleep 0.1
    done
fi
exec /bin/k3s "$@"
`

type K3sSpec struct {
	ID               string
	Logf             func(format string, arguments ...any)
	ObserveResources func(K3sResources)
	// Lifecycle tests use the options below to produce one missing prerequisite each.
	ShouldRunUnprivileged       bool
	ShouldUseUnreachableAddress bool
	ExtraImages                 []K3sImage
}

type K3sResources struct {
	ContainerIDs []string
	NetworkName  string
	VolumeNames  []string
}

// K3s is one disposable Kubernetes cluster in a privileged Docker container. It is driven only
// through kubectl inside the container: no API port is published and no kubeconfig is written on
// the host, so the developer's Kubernetes configuration is never read or changed.
type K3s struct {
	ID          string
	Address     string
	NetworkName string
	VolumeName  string
	docker      *testcontainers.DockerClient
	container   testcontainers.Container
	network     *testcontainers.DockerNetwork
	logf        func(format string, arguments ...any)
}

// StartK3s starts k3s with the pinned images and the CloudNativePG operator. On any failure it removes
// what it created and returns the cause with bounded diagnostics.
func StartK3s(ctx context.Context, spec K3sSpec) (*K3s, error) {
	fixture := &K3s{ID: spec.ID, logf: spec.Logf}
	if fixture.ID == "" {
		fixture.ID = uuid.NewString()
	}

	if fixture.logf == nil {
		fixture.logf = func(string, ...any) {}
	}

	startedAt := time.Now()
	// Every startup step has its own limit, so on a CPU-starved host the steps add up to a wait of
	// many minutes before anything fails. One limit for the whole startup reports that host early.
	startCtx, cancel := context.WithTimeoutCause(ctx, k3sTotalStartLimit, ErrK3sStartTimedOut)
	defer cancel()

	err := fixture.start(startCtx, spec)
	if err != nil && errors.Is(context.Cause(startCtx), ErrK3sStartTimedOut) {
		err = fmt.Errorf("%w within %s: %w", ErrK3sStartTimedOut, k3sTotalStartLimit, err)
	}

	if err != nil {
		diagnostics := fixture.getDiagnostics(context.WithoutCancel(ctx))
		terminateErr := fixture.Terminate(context.WithoutCancel(ctx))

		return nil, errors.Join(fmt.Errorf("start k3s fixture %s: %w\n%s", fixture.ID, err, diagnostics), terminateErr)
	}

	fixture.logf("k3s fixture %s ready at %s after %s", fixture.ID, fixture.Address, time.Since(startedAt))

	return fixture, nil
}

// Terminate removes the container, its volume and its network, in that order, because the volume
// and the network are in use while the container exists.
func (fixture *K3s) Terminate(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, k3sTerminateLimit)
	defer cancel()

	var errs []error
	if fixture.container != nil {
		errs = append(errs, fixture.container.Terminate(ctx, testcontainers.StopTimeout(0)))
	}

	if fixture.VolumeName != "" && fixture.docker != nil {
		_, err := fixture.docker.VolumeRemove(ctx, fixture.VolumeName, client.VolumeRemoveOptions{Force: true})
		errs = append(errs, err)
	}

	if fixture.network != nil {
		errs = append(errs, fixture.network.Remove(ctx))
	}

	if fixture.docker != nil {
		errs = append(errs, fixture.docker.Close())
	}

	return errors.Join(errs...)
}

func (fixture *K3s) start(ctx context.Context, spec K3sSpec) error {
	if err := requireDockerFromEnvironment(ctx); err != nil {
		return err
	}

	docker, err := testcontainers.NewDockerClientWithOpts(ctx)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrDockerUnavailable, err)
	}

	fixture.docker = docker
	if _, err := fixture.docker.Ping(ctx, client.PingOptions{}); err != nil {
		return fmt.Errorf("%w: %w", ErrDockerUnavailable, err)
	}

	labels, err := addTestRunLabel(map[string]string{LabelK3sFixture: fixture.ID})
	if err != nil {
		return err
	}

	images, err := getCNPGImages()
	if err != nil {
		return err
	}

	images = append(images, spec.ExtraImages...)

	if err := fixture.createNetwork(ctx, labels); err != nil {
		return err
	}

	fixture.observe(ctx, spec)

	volume, err := fixture.docker.VolumeCreate(
		ctx,
		client.VolumeCreateOptions{Name: "databasus-k3s-" + fixture.ID, Labels: labels},
	)
	if err != nil {
		return fmt.Errorf("create k3s volume: %w", err)
	}

	fixture.VolumeName = volume.Volume.Name
	fixture.observe(ctx, spec)

	if err := fixture.startContainer(
		ctx,
		k3sContainerSpec{Labels: labels, IsPrivileged: !spec.ShouldRunUnprivileged},
	); err != nil {
		return err
	}

	fixture.observe(ctx, spec)

	if err := fixture.waitForNode(ctx); err != nil {
		return err
	}

	if err := fixture.requireBridgeNetfilter(ctx); err != nil {
		return err
	}

	if err := fixture.importImages(ctx, images); err != nil {
		return err
	}

	if spec.ShouldUseUnreachableAddress {
		fixture.Address = getUnassignedAddress(fixture.Address)
	}

	if err := fixture.requireReachableAddress(ctx); err != nil {
		return err
	}

	if err := fixture.requireVersion(ctx); err != nil {
		return err
	}

	if err := fixture.waitForSystemComponents(ctx); err != nil {
		return err
	}

	return fixture.installCNPGOperator(ctx)
}

func (fixture *K3s) createNetwork(ctx context.Context, labels map[string]string) error {
	return WithFreeSubnet(func(ipam *dockernetwork.IPAM) error {
		createdNetwork, err := network.New(ctx, network.WithLabels(labels), network.WithIPAM(ipam))
		if err != nil {
			return err
		}

		fixture.network = createdNetwork
		fixture.NetworkName = createdNetwork.Name

		return nil
	})
}

type k3sContainerSpec struct {
	Labels       map[string]string
	IsPrivileged bool
}

func (fixture *K3s) startContainer(ctx context.Context, spec k3sContainerSpec) error {
	image, err := getCatalogReference(k3sImage)
	if err != nil {
		return err
	}

	if err := fixture.pullImage(ctx, image); err != nil {
		return err
	}

	request := testcontainers.ContainerRequest{
		Image:      image,
		Labels:     spec.Labels,
		Networks:   []string{fixture.NetworkName},
		Entrypoint: []string{"/bin/sh", k3sEntrypointFile},
		Cmd: []string{
			"server",
			"--disable=traefik,servicelb,metrics-server",
			"--disable-helm-controller",
			"--disable-network-policy",
			"--disable-default-registry-endpoint",
			"--private-registry=" + k3sRegistriesFile,
			"--kubelet-arg=max-pods=" + strconv.Itoa(k3sMaxPods),
		},
		Files: []testcontainers.ContainerFile{
			{Reader: strings.NewReader(k3sRegistries), ContainerFilePath: k3sRegistriesFile, FileMode: 0o600},
			{Reader: strings.NewReader(k3sEntrypoint), ContainerFilePath: k3sEntrypointFile, FileMode: 0o755},
		},
		HostConfigModifier: func(host *dockercontainer.HostConfig) {
			host.Privileged = spec.IsPrivileged
			// The image declares these as volumes; without a mount Docker would create anonymous
			// volumes that carry no fixture or run label and survive a killed run.
			host.Tmpfs = map[string]string{
				"/var/lib/kubelet": "rw,size=1g",
				"/var/lib/cni":     "rw,size=64m",
				"/var/log":         "rw,size=256m",
			}
			host.Mounts = append(host.Mounts, mount.Mount{
				Type:   mount.TypeVolume,
				Source: fixture.VolumeName,
				Target: k3sDataDirectory,
			})
		},
	}

	container, err := testcontainers.GenericContainer(
		ctx,
		testcontainers.GenericContainerRequest{ContainerRequest: request, Started: true},
	)
	if container != nil {
		fixture.container = container
	}

	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "privileged") {
			return fmt.Errorf("%w: %w", ErrPrivilegedContainerRequired, err)
		}

		return fmt.Errorf("start k3s container: %w", err)
	}

	address, err := container.ContainerIP(ctx)
	if err != nil {
		return fmt.Errorf("read k3s container address: %w", err)
	}

	fixture.Address = address

	return nil
}

// waitForNode waits until the Kubernetes node is Ready. k3s brings containerd up and reconfigures it
// before that, and an image import into a containerd that is still starting fails mid-stream.
func (fixture *K3s) waitForNode(ctx context.Context) error {
	return fixture.waitFor(ctx, "the Kubernetes node is ready", func() error {
		output, err := fixture.Kubectl(ctx, KubectlCommand{Arguments: []string{
			"get", "nodes", "-o", `jsonpath={.items[*].status.conditions[?(@.type=="Ready")].status}`,
		}})
		if err != nil {
			return err
		}

		if strings.TrimSpace(string(output)) != "True" {
			return fmt.Errorf("node ready condition is %q", output)
		}

		return nil
	})
}

// requireReachableAddress completes a TLS handshake with the API server verified against this
// k3s's own authority, so a host that accepts any connection on the address cannot pass for it.
func (fixture *K3s) requireReachableAddress(ctx context.Context) error {
	output, err := fixture.exec(ctx, dockerExec{Command: []string{"cat", k3sServerAuthorityFile}})
	if err != nil || output.ExitCode != 0 {
		return fmt.Errorf("read the k3s server authority: %w: %s", err, output.Stderr)
	}

	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(output.Stdout) {
		return fmt.Errorf("parse the k3s server authority")
	}

	dialer := tls.Dialer{
		NetDialer: &net.Dialer{Timeout: k3sAddressDialLimit},
		Config:    &tls.Config{RootCAs: roots, ServerName: fixture.Address, MinVersion: tls.VersionTLS12},
	}
	connection, err := dialer.DialContext(
		ctx,
		"tcp",
		net.JoinHostPort(fixture.Address, strconv.Itoa(k3sAPIPort)),
	)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrContainerAddressUnreachable, err)
	}

	return connection.Close()
}

// Without br_netfilter on the host, a pod's request to a service address on the same node bypasses
// the service rules, so cluster DNS times out and CloudNativePG never joins its replica. k3s cannot
// load the module from inside its container.
func (fixture *K3s) requireBridgeNetfilter(ctx context.Context) error {
	output, err := fixture.exec(ctx, dockerExec{Command: []string{
		"sh", "-c", "test -e /proc/sys/net/bridge/bridge-nf-call-iptables && echo present || echo absent",
	}})
	if err != nil {
		return err
	}

	if strings.TrimSpace(string(output.Stdout)) != "present" {
		return ErrBridgeNetfilterRequired
	}

	return nil
}

func (fixture *K3s) requireVersion(ctx context.Context) error {
	output, err := fixture.exec(ctx, dockerExec{Command: []string{"k3s", "--version"}})
	if err != nil {
		return err
	}

	if !strings.Contains(string(output.Stdout), k3sVersion) {
		return fmt.Errorf("k3s reports %q, expected %s", output.Stdout, k3sVersion)
	}

	fixture.logf("k3s fixture %s version: %s", fixture.ID, strings.SplitN(string(output.Stdout), "\n", 2)[0])

	return nil
}

// The system pods may have been created before their images were imported; a pod still waiting for
// an image is recreated, so it does not sit out the kubelet's pull back-off.
func (fixture *K3s) waitForSystemComponents(ctx context.Context) error {
	for _, deployment := range []string{"coredns", "local-path-provisioner"} {
		if err := fixture.waitFor(ctx, "k3s creates "+deployment, func() error {
			_, err := fixture.Kubectl(ctx, KubectlCommand{
				Arguments: []string{"-n", "kube-system", "get", "deployment", deployment},
			})

			return err
		}); err != nil {
			return err
		}
	}

	if _, err := fixture.Kubectl(ctx, KubectlCommand{Arguments: []string{
		"-n", "kube-system", "delete", "pods", "--field-selector=status.phase=Pending", "--wait=false",
	}}); err != nil {
		return err
	}

	for _, deployment := range []string{"coredns", "local-path-provisioner"} {
		if _, err := fixture.Kubectl(ctx, KubectlCommand{Arguments: []string{
			"-n", "kube-system", "rollout", "status", "deployment/" + deployment,
			"--timeout=" + k3sStartupLimit.String(),
		}}); err != nil {
			return err
		}
	}

	return nil
}

func (fixture *K3s) waitFor(ctx context.Context, condition string, check func() error) error {
	deadline := time.Now().Add(k3sStartupLimit)
	for {
		err := check()
		if err == nil {
			return nil
		}

		if exitErr := fixture.getExitError(ctx); exitErr != nil {
			return exitErr
		}

		if time.Now().After(deadline) {
			return fmt.Errorf("%s within %s: %w", condition, k3sStartupLimit, err)
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("%s: %w", condition, ctx.Err())
		case <-time.After(k3sPollInterval):
		}
	}
}

// k3s needs to write cgroups and mounts; without a privileged container it exits at once with a
// permission or read-only file system error.
func (fixture *K3s) getExitError(ctx context.Context) error {
	if fixture.container == nil {
		return nil
	}

	state, err := fixture.container.State(ctx)
	if err != nil || state.Running {
		return nil
	}

	logs := fixture.getLogTail(ctx)
	inspection, inspectErr := fixture.container.Inspect(ctx)
	isPrivileged := inspectErr == nil && inspection.HostConfig != nil && inspection.HostConfig.Privileged
	if !isPrivileged || strings.Contains(logs, "read-only file system") || strings.Contains(logs, "permission denied") {
		return fmt.Errorf("%w: k3s exited with code %d: %s", ErrPrivilegedContainerRequired, state.ExitCode, logs)
	}

	return fmt.Errorf("k3s exited with code %d: %s", state.ExitCode, logs)
}

func (fixture *K3s) observe(ctx context.Context, spec K3sSpec) {
	if spec.ObserveResources == nil {
		return
	}

	resources := K3sResources{NetworkName: fixture.NetworkName}
	if fixture.VolumeName != "" {
		resources.VolumeNames = append(resources.VolumeNames, fixture.VolumeName)
	}

	if fixture.container != nil {
		resources.ContainerIDs = append(resources.ContainerIDs, fixture.container.GetContainerID())
		if inspection, err := fixture.container.Inspect(ctx); err == nil {
			for _, containerMount := range inspection.Mounts {
				if containerMount.Name != "" && containerMount.Name != fixture.VolumeName {
					resources.VolumeNames = append(resources.VolumeNames, containerMount.Name)
				}
			}
		}
	}

	spec.ObserveResources(resources)
}

// requireDockerFromEnvironment reaches the daemon DOCKER_HOST names, or the default socket. Docker
// discovery in testcontainers falls back to other sockets when that daemon is unreachable, which
// would hide a missing prerequisite behind another daemon.
func requireDockerFromEnvironment(ctx context.Context) error {
	docker, err := client.New(client.FromEnv)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrDockerUnavailable, err)
	}
	defer func() { _ = docker.Close() }()

	if _, err := docker.Ping(ctx, client.PingOptions{}); err != nil {
		return fmt.Errorf("%w: %w", ErrDockerUnavailable, err)
	}

	return nil
}

// getUnassignedAddress returns the last host address of the fixture's own /24, which no container
// holds, for the lifecycle test of an unreachable address.
func getUnassignedAddress(address string) string {
	octets := netip.MustParseAddr(address).As4()
	octets[3] = 254

	return netip.AddrFrom4(octets).String()
}
