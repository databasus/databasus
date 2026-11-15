package containers

import (
	"context"
	"net/netip"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const cleanupClockShift = abandonedNetworkAge + time.Minute

func TestRemoveAbandonedNetworks(t *testing.T) {
	t.Parallel()

	t.Run("removes only old test networks without running containers", func(t *testing.T) {
		t.Parallel()
		docker := newCleanupDockerHost(t)
		emptyNetworkName := createTestRangeNetwork(t, docker)
		runningNetworkName := createTestRangeNetwork(t, docker)
		startProbeContainer(t, docker, runningNetworkName, []string{"sleep", "600"})
		stoppedNetworkName := createTestRangeNetwork(t, docker)
		stoppedContainerName := startProbeContainer(t, docker, stoppedNetworkName, []string{"true"})
		_, err := docker.WaitForExit(t.Context(), stoppedContainerName, time.Minute)
		require.NoError(t, err)
		defaultPoolNetworkName := createDefaultPoolNetwork(t, docker)
		probeNetworkNames := []string{emptyNetworkName, runningNetworkName, stoppedNetworkName, defaultPoolNetworkName}
		scope := networkCleanupScope{now: time.Now().Add(cleanupClockShift), networkNames: probeNetworkNames}

		err = removeAbandonedNetworks(t.Context(), docker.GetClient(), scope)

		require.NoError(t, err)
		remainingNetworkNames, err := docker.GetRemainingResources(
			t.Context(),
			DockerResources{ContainerNames: nil, NetworkNames: probeNetworkNames, VolumeNames: nil},
		)
		require.NoError(t, err)
		assert.ElementsMatch(t, []string{runningNetworkName, defaultPoolNetworkName}, remainingNetworkNames)
	})

	t.Run("keeps an empty test network when it is younger than an hour", func(t *testing.T) {
		t.Parallel()
		docker := newCleanupDockerHost(t)
		freshNetworkName := createTestRangeNetwork(t, docker)
		scope := networkCleanupScope{now: time.Now(), networkNames: []string{freshNetworkName}}

		err := removeAbandonedNetworks(t.Context(), docker.GetClient(), scope)

		require.NoError(t, err)
		remainingNetworkNames, err := docker.GetRemainingResources(
			t.Context(),
			DockerResources{ContainerNames: nil, NetworkNames: []string{freshNetworkName}, VolumeNames: nil},
		)
		require.NoError(t, err)
		assert.Equal(t, []string{freshNetworkName}, remainingNetworkNames)
	})
}

func TestIsAbandonedTestNetwork(t *testing.T) {
	t.Parallel()

	now := time.Now()
	oldCreation := now.Add(-cleanupClockShift)
	testRangeSubnet := netip.MustParsePrefix("10.170.5.0/24")
	defaultPoolSubnet := netip.MustParsePrefix("172.20.0.0/16")

	testCases := []struct {
		name        string
		createdAt   time.Time
		subnets     []netip.Prefix
		isAbandoned bool
	}{
		{
			name:        "accepts an old network when its only subnet is in the test range",
			createdAt:   oldCreation,
			subnets:     []netip.Prefix{testRangeSubnet},
			isAbandoned: true,
		},
		{
			name:        "rejects an old network without a subnet",
			createdAt:   oldCreation,
			subnets:     nil,
			isAbandoned: false,
		},
		{
			name:        "rejects an old network with a subnet outside the test range",
			createdAt:   oldCreation,
			subnets:     []netip.Prefix{testRangeSubnet, defaultPoolSubnet},
			isAbandoned: false,
		},
		{
			name:        "rejects a test range network when it is younger than an hour",
			createdAt:   now.Add(-time.Minute),
			subnets:     []netip.Prefix{testRangeSubnet},
			isAbandoned: false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			var ipamConfigs []network.IPAMConfig
			for _, subnet := range testCase.subnets {
				ipamConfigs = append(ipamConfigs, network.IPAMConfig{Subnet: subnet})
			}
			listedNetwork := network.Summary{Created: testCase.createdAt, IPAM: network.IPAM{Config: ipamConfigs}}

			isAbandoned := isAbandonedTestNetwork(listedNetwork, now)

			assert.Equal(t, testCase.isAbandoned, isAbandoned)
		})
	}
}

func newCleanupDockerHost(t *testing.T) *DockerHost {
	t.Helper()

	docker, err := NewDockerHost()
	require.NoError(t, err)
	t.Cleanup(docker.Close)

	return docker
}

func createTestRangeNetwork(t *testing.T, docker *DockerHost) string {
	t.Helper()
	networkName := newCleanupProbeName()

	require.NoError(t, docker.CreateNetwork(t.Context(), networkName))
	t.Cleanup(func() { assert.NoError(t, docker.RemoveNetwork(context.WithoutCancel(t.Context()), networkName)) })

	return networkName
}

func createDefaultPoolNetwork(t *testing.T, docker *DockerHost) string {
	t.Helper()
	networkName := newCleanupProbeName()

	_, err := docker.GetClient().NetworkCreate(
		t.Context(),
		networkName,
		client.NetworkCreateOptions{Labels: docker.GetLabels()},
	)
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, docker.RemoveNetwork(context.WithoutCancel(t.Context()), networkName)) })

	return networkName
}

func startProbeContainer(t *testing.T, docker *DockerHost, networkName string, entrypoint []string) string {
	t.Helper()
	image, err := getCatalogReference("alpine:3.21")
	require.NoError(t, err)
	containerName := newCleanupProbeName()

	err = docker.StartContainer(t.Context(), RunSpec{
		Name:          containerName,
		Image:         image,
		Entrypoint:    entrypoint,
		Env:           nil,
		Mount:         "",
		NetworkName:   networkName,
		NetworkAlias:  containerName,
		PublishedPort: "",
	})
	t.Cleanup(func() { assert.NoError(t, docker.RemoveContainer(context.WithoutCancel(t.Context()), containerName)) })
	require.NoError(t, err)

	return containerName
}

func newCleanupProbeName() string {
	return "databasus-network-cleanup-" + uuid.NewString()
}
