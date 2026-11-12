package containers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/netip"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/moby/moby/api/pkg/stdcopy"
	dockercontainer "github.com/moby/moby/api/types/container"
	dockernetwork "github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/network"
	"github.com/testcontainers/testcontainers-go/wait"
)

const LabelPatroniFixture = "databasus.patroni-fixture"

type Patroni struct {
	Endpoint           Endpoint
	Nodes              [2]PatroniNode
	ID                 string
	NetworkName        string
	TemporaryDirectory string
	containers         []testcontainers.Container
	observeResources   func(PatroniResources)
}

type PatroniNode struct {
	Name      string
	SQL       Endpoint
	API       Endpoint
	Address   string
	Container testcontainers.Container
}

type PatroniSpec struct {
	Image                   string
	ID                      string
	NodeEntrypoint          []string
	ObserveResources        func(PatroniResources)
	ShouldUseExternalConfig bool
}

type PatroniResources struct {
	NetworkName        string
	ContainerIDs       []string
	VolumeNames        []string
	TemporaryDirectory string
}

func StartPatroni(t *testing.T, spec PatroniSpec) *Patroni {
	t.Helper()

	startedAt := time.Now()
	fixture := &Patroni{ID: spec.ID, TemporaryDirectory: t.TempDir(), observeResources: spec.ObserveResources}
	if fixture.ID == "" {
		fixture.ID = uuid.NewString()
	}

	labels := withTestRunLabel(t, map[string]string{LabelPatroniFixture: fixture.ID})
	t.Logf("fixture %s creating network", fixture.ID)
	var fixtureNetwork *testcontainers.DockerNetwork
	err := WithFreeSubnet(func(ipam *dockernetwork.IPAM) error {
		var networkErr error
		fixtureNetwork, networkErr = network.New(t.Context(), network.WithLabels(labels), network.WithIPAM(ipam))
		return networkErr
	})
	t.Logf("fixture %s created network after %s: %v", fixture.ID, time.Since(startedAt), err)
	require.NoError(t, err)
	fixture.NetworkName = fixtureNetwork.Name
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.WithoutCancel(t.Context()), time.Minute)
		defer cancel()
		for _, container := range slices.Backward(fixture.containers) {
			if t.Failed() {
				fixture.logDiagnostics(ctx, t, container)
			}
			assert.NoError(
				t,
				container.Terminate(ctx, testcontainers.StopTimeout(0)),
				"remove fixture container %s",
				container.GetContainerID(),
			)
		}
		assert.NoError(t, fixtureNetwork.Remove(ctx), "remove fixture network %s", fixture.NetworkName)
	})
	fixture.observe(t)

	fixture.start(t, testcontainers.ContainerRequest{
		Image: GetPinnedImage(t, "quay.io/coreos/etcd:v3.6.16"),
		Cmd: []string{
			"/usr/local/bin/etcd", "--name=etcd", "--data-dir=/etcd-data",
			"--listen-client-urls=http://0.0.0.0:2379", "--advertise-client-urls=http://etcd:2379",
			"--listen-peer-urls=http://0.0.0.0:2380", "--initial-advertise-peer-urls=http://etcd:2380",
			"--initial-cluster=etcd=http://etcd:2380",
		},
		NetworkAliases: map[string][]string{fixture.NetworkName: {"etcd"}},
		WaitingFor:     wait.ForLog("ready to serve client requests").WithStartupTimeout(2 * time.Minute),
	}, labels)

	baseImage := GetPinnedImage(t, spec.Image)
	for index, name := range []string{"node-a", "node-b"} {
		settings := patroniConfiguration(
			patroniConfigurationSpec{
				Scope:                   fixture.ID,
				Name:                    name,
				ShouldUseExternalConfig: spec.ShouldUseExternalConfig,
			},
		)
		configuration, encodeErr := json.Marshal(settings)
		require.NoError(t, encodeErr)
		configurationPath := filepath.Join(fixture.TemporaryDirectory, name+".json")
		require.NoError(t, os.WriteFile(configurationPath, configuration, 0o644))
		node := fixture.start(t, testcontainers.ContainerRequest{
			Context: getFixtureDirectory(t, "patroni"),
			Repo:    "databasus-test-patroni", Tag: restoreTargetTag(spec.Image), KeepImage: true,
			BuildArgs:    map[string]*string{"POSTGRES_IMAGE": &baseImage},
			ExposedPorts: []string{"5432/tcp", "8008/tcp"},
			Entrypoint:   spec.NodeEntrypoint,
			Files: []testcontainers.ContainerFile{
				{HostFilePath: configurationPath, ContainerFilePath: "/etc/patroni.json", FileMode: 0o644},
			},
			NetworkAliases: map[string][]string{fixture.NetworkName: {name}},
			WaitingFor:     wait.ForHTTP("/health").WithPort("8008/tcp").WithStartupTimeout(3 * time.Minute),
		}, labels)
		address, addressErr := node.ContainerIP(t.Context())
		require.NoError(t, addressErr)
		fixture.Nodes[index] = PatroniNode{
			Name:      name,
			SQL:       endpointOf(t, node, "5432/tcp"),
			API:       endpointOf(t, node, "8008/tcp"),
			Address:   address,
			Container: node,
		}
	}

	proxy := fixture.start(t, testcontainers.ContainerRequest{
		Image:        "databasus-test-patroni:" + restoreTargetTag(spec.Image),
		Entrypoint:   []string{"haproxy", "-W", "-db", "-f", "/etc/haproxy.cfg"},
		ExposedPorts: []string{"5432/tcp"},
		Files: []testcontainers.ContainerFile{
			{
				Reader:            strings.NewReader(patroniProxyConfiguration()),
				ContainerFilePath: "/etc/haproxy.cfg",
				FileMode:          0o644,
			},
		},
		User:       "postgres",
		WaitingFor: wait.ForListeningPort("5432/tcp"),
	}, labels)
	fixture.Endpoint = endpointOf(t, proxy, "5432/tcp")
	for _, binary := range []struct {
		container testcontainers.Container
		command   []string
		version   string
	}{
		{container: fixture.containers[0], command: []string{"/usr/local/bin/etcd", "--version"}, version: "3.6.16"},
		{container: fixture.Nodes[0].Container, command: []string{"patroni", "--version"}, version: "4.1.5"},
		{container: proxy, command: []string{"haproxy", "-v"}, version: "3.2.26"},
	} {
		code, output, err := binary.container.Exec(t.Context(), binary.command)
		require.NoError(t, err)
		body, err := io.ReadAll(io.LimitReader(output, 4096))
		require.NoError(t, err)
		require.Zero(t, code)
		require.Contains(t, string(body), binary.version)
		t.Logf("fixture version: %s", strings.TrimSpace(string(body)))
	}

	connection := fixture.Connect(t, fixture.Endpoint)
	_, err = connection.Exec(t.Context(), "CREATE DATABASE testdb")
	require.NoError(t, err)
	t.Logf("fixture %s waiting for cluster readiness after %s", fixture.ID, time.Since(startedAt))
	fixture.RequireReady(t)

	if spec.ShouldUseExternalConfig {
		for _, node := range fixture.Nodes {
			connection := fixture.Connect(t, node.SQL)
			for _, setting := range []string{"config_file", "hba_file", "ident_file"} {
				var path string
				require.NoError(t, connection.QueryRow(t.Context(), "SELECT current_setting($1)", setting).Scan(&path))
				require.True(t, strings.HasPrefix(path, "/var/lib/postgresql/configuration/"), "%s: %s", setting, path)
			}
			code, _, err := node.Container.Exec(t.Context(), []string{
				"rm", "-f",
				"/var/lib/postgresql/patroni/postgresql.conf", "/var/lib/postgresql/patroni/postgresql.base.conf",
				"/var/lib/postgresql/patroni/pg_hba.conf", "/var/lib/postgresql/patroni/pg_ident.conf",
			})
			require.NoError(t, err)
			require.Zero(t, code)
		}
	}

	t.Logf("fixture %s ready after %s", fixture.ID, time.Since(startedAt))

	return fixture
}

func (fixture *Patroni) start(
	t *testing.T,
	request testcontainers.ContainerRequest,
	labels map[string]string,
) testcontainers.Container {
	t.Helper()

	request.Labels = labels
	request.Networks = []string{fixture.NetworkName}
	request.HostConfigModifier = func(host *dockercontainer.HostConfig) {
		for port, bindings := range host.PortBindings {
			for index := range bindings {
				bindings[index].HostIP = netip.MustParseAddr(loopbackIP)
			}
			host.PortBindings[port] = bindings
		}
	}

	containerName := getPatroniContainerName(request, fixture.NetworkName)
	startedAt := time.Now()
	t.Logf("fixture %s starting %s", fixture.ID, containerName)
	container, err := testcontainers.GenericContainer(
		t.Context(),
		testcontainers.GenericContainerRequest{ContainerRequest: request, Started: true},
	)
	t.Logf("fixture %s started %s after %s: %v", fixture.ID, containerName, time.Since(startedAt), err)
	if container != nil {
		fixture.containers = append(fixture.containers, container)
	}
	fixture.observe(t)
	require.NoError(t, err, "start Patroni fixture %s", fixture.ID)

	return container
}

func getPatroniContainerName(request testcontainers.ContainerRequest, networkName string) string {
	if aliases := request.NetworkAliases[networkName]; len(aliases) > 0 {
		return aliases[0]
	}

	if len(request.Entrypoint) > 0 {
		return request.Entrypoint[0]
	}

	return request.Image
}

func (fixture *Patroni) observe(t *testing.T) {
	t.Helper()

	if fixture.observeResources == nil {
		return
	}

	resources := PatroniResources{NetworkName: fixture.NetworkName, TemporaryDirectory: fixture.TemporaryDirectory}
	for _, container := range fixture.containers {
		resources.ContainerIDs = append(resources.ContainerIDs, container.GetContainerID())
		inspection, err := container.Inspect(t.Context())
		require.NoError(t, err)

		for _, mount := range inspection.Mounts {
			if mount.Name != "" {
				resources.VolumeNames = append(resources.VolumeNames, mount.Name)
			}
		}
	}

	fixture.observeResources(resources)
}

func (fixture *Patroni) logDiagnostics(ctx context.Context, t *testing.T, container testcontainers.Container) {
	t.Helper()

	docker, err := testcontainers.NewDockerClientWithOpts(ctx)
	if err != nil {
		t.Logf("fixture diagnostics unavailable: %v", err)
		return
	}
	defer func() { _ = docker.Close() }()
	logs, err := docker.ContainerLogs(
		ctx,
		container.GetContainerID(),
		client.ContainerLogsOptions{ShowStdout: true, ShowStderr: true, Tail: "40"},
	)
	if err != nil {
		t.Logf("fixture %s diagnostics: %v", fixture.ID, err)
		return
	}
	defer func() { _ = logs.Close() }()
	var output bytes.Buffer
	_, err = stdcopy.StdCopy(&output, &output, io.LimitReader(logs, 64*1024))
	if err != nil {
		t.Logf("fixture %s diagnostics truncated: %v", fixture.ID, err)
	}
	tail := output.Bytes()
	if len(tail) > failedContainerOutputTailBytes {
		tail = tail[len(tail)-failedContainerOutputTailBytes:]
	}
	text := strings.ReplaceAll(string(tail), PostgresPassword, "[REDACTED]")
	t.Logf("fixture %s container %s: %s", fixture.ID, container.GetContainerID(), text)
}

type patroniConfigurationSpec struct {
	Scope                   string
	Name                    string
	ShouldUseExternalConfig bool
}

func patroniConfiguration(spec patroniConfigurationSpec) map[string]any {
	name := spec.Name
	configurationDirectory := "/var/lib/postgresql/patroni"
	if spec.ShouldUseExternalConfig {
		configurationDirectory = "/var/lib/postgresql/configuration"
	}

	return map[string]any{
		"scope": spec.Scope, "namespace": "/databasus", "name": name,
		"restapi": map[string]any{"listen": "0.0.0.0:8008", "connect_address": name + ":8008"},
		"etcd3":   map[string]any{"hosts": []string{"etcd:2379"}},
		"bootstrap": map[string]any{
			"initdb": []any{map[string]string{"encoding": "UTF8"}, "data-checksums"},
			"dcs": map[string]any{
				"loop_wait": 2, "retry_timeout": 3, "ttl": 20, "primary_start_timeout": 0,
				"postgresql": map[string]any{"use_pg_rewind": true, "parameters": map[string]any{
					"fsync":                 "on",
					"full_page_writes":      "on",
					"synchronous_commit":    "on",
					"wal_level":             "replica",
					"summarize_wal":         "on",
					"max_wal_senders":       16,
					"max_replication_slots": 16,
					"wal_keep_size":         "64MB",
					"max_wal_size":          "256MB",
					"min_wal_size":          "32MB",
				}},
			},
		},
		"postgresql": map[string]any{
			"listen":          "0.0.0.0:5432",
			"connect_address": name + ":5432",
			"data_dir":        "/var/lib/postgresql/patroni",
			"config_dir":      configurationDirectory,
			"authentication": map[string]any{
				"superuser":   map[string]string{"username": PostgresUsername, "password": PostgresPassword},
				"replication": map[string]string{"username": "replicator", "password": PostgresPassword},
			},
			"pg_hba": []string{
				"local all all trust",
				"host all all 0.0.0.0/0 scram-sha-256",
				"host replication all 0.0.0.0/0 scram-sha-256",
			},
		},
	}
}

func patroniProxyConfiguration() string {
	return fmt.Sprintf(`global
  maxconn 256
defaults
  mode tcp
  timeout connect 3s
  timeout client 10m
  timeout server 10m
frontend postgres
  bind *:5432
  default_backend primary
backend primary
  option httpchk GET /primary
  http-check expect status 200
  default-server inter 200ms fall 1 rise 1 on-marked-down shutdown-sessions
  server node-a node-a:5432 check port %d
  server node-b node-b:5432 check port %d
`, 8008, 8008)
}
