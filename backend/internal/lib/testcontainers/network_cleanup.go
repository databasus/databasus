package containers

import (
	"context"
	"fmt"
	"log/slog"
	// Shuffling the networks to remove contains no secrets.
	// nosemgrep: go.lang.security.audit.crypto.math_random.math-random-used
	"math/rand/v2"
	"slices"
	"sync"
	"time"

	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"

	"databasus-backend/internal/lib/background"
)

const (
	// No test process lives this long: the longest go test -timeout in the repository is 40 minutes.
	// Docker removes a network whose containers are all stopped, so the age alone protects a live
	// test that stopped its container for a while. A process run with -timeout 0 or under a debugger
	// for over an hour can lose such a network.
	abandonedNetworkAge = time.Hour

	// A sweep in a process that no test timeout bounds, such as one run with -timeout 0, stops after
	// this and leaves the rest to the next process.
	abandonedNetworkCleanupLimit = 15 * time.Minute
)

//nolint:gochecknoglobals // Once-per-process abandoned network cleanup coordinates the test fixtures of one process.
var abandonedNetworksCleanup sync.Once

type networkCleanupScope struct {
	now time.Time
	// An empty list puts every test network in scope.
	networkNames []string
}

// A process killed before its cleanups ran, or a Docker restart, leaves test networks behind. With
// Docker on firewalld every network is an interface in the firewall zone, and firewalld slows with
// each one until creating a network outlasts test timeouts. Every process removes them once, in the
// background: behind a slow firewalld removing a large pile takes minutes, longer than the first
// test of the process may wait for its own network.
func removeAbandonedNetworksOncePerProcess(ctx context.Context) {
	abandonedNetworksCleanup.Do(func() {
		cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), abandonedNetworkCleanupLimit)
		goroutines := background.New(background.Deps{Logger: slog.New(slog.DiscardHandler)})

		goroutines.Go(cleanupCtx, "abandoned test network cleanup", func(ctx context.Context) error {
			defer cancel()

			docker, err := client.New(client.FromEnv)
			if err != nil {
				return fmt.Errorf("connect to Docker: %w", err)
			}
			defer func() { _ = docker.Close() }()

			return removeAbandonedNetworks(ctx, docker, networkCleanupScope{now: time.Now(), networkNames: nil})
		})
	})
}

// removeAbandonedNetworks removes the test networks in scope that were created over an hour before
// the scope's time. It takes no lock and inspects no network: Docker refuses to remove a network with
// a running container, and a neighbouring process may remove the same network first, so every
// removal error is ignored. The shuffle sends concurrent processes to different networks first.
func removeAbandonedNetworks(ctx context.Context, docker *client.Client, scope networkCleanupScope) error {
	listed, err := docker.NetworkList(ctx, client.NetworkListOptions{})
	if err != nil {
		return fmt.Errorf("list networks: %w", err)
	}

	var abandonedNetworkIDs []string
	for _, listedNetwork := range listed.Items {
		if len(scope.networkNames) > 0 && !slices.Contains(scope.networkNames, listedNetwork.Name) {
			continue
		}

		if isAbandonedTestNetwork(listedNetwork, scope.now) {
			abandonedNetworkIDs = append(abandonedNetworkIDs, listedNetwork.ID)
		}
	}

	rand.Shuffle(len(abandonedNetworkIDs), func(i, j int) {
		abandonedNetworkIDs[i], abandonedNetworkIDs[j] = abandonedNetworkIDs[j], abandonedNetworkIDs[i]
	})

	for _, networkID := range abandonedNetworkIDs {
		_, _ = docker.NetworkRemove(ctx, networkID, client.NetworkRemoveOptions{})
	}

	return nil
}

// isAbandonedTestNetwork requires at least one subnet, because "every subnet is in the test range"
// holds trivially for host, none and other networks without address management.
func isAbandonedTestNetwork(listedNetwork network.Summary, now time.Time) bool {
	if len(listedNetwork.IPAM.Config) == 0 || now.Sub(listedNetwork.Created) < abandonedNetworkAge {
		return false
	}

	for _, config := range listedNetwork.IPAM.Config {
		if !IsTestNetworkSubnet(config.Subnet) {
			return false
		}
	}

	return true
}
