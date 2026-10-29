package container

import (
	"context"
	"errors"
	"fmt"

	cerrdefs "github.com/containerd/errdefs"
)

// leftoverEngine is the Docker seam for removing labeled leftovers — the
// methods the unit tests fake, rather than the whole Docker surface.
type leftoverEngine interface {
	ListManagedContainers(ctx context.Context, labels map[string]string) ([]ManagedResource, error)
	ListManagedNetworks(ctx context.Context, labels map[string]string) ([]ManagedResource, error)
	RemoveContainer(ctx context.Context, containerID string) error
	RemoveNetwork(ctx context.Context, networkID string) error
}

type attemptNetworkEngine interface {
	leftoverEngine
	CreateNetwork(ctx context.Context, name string, labels map[string]string) (string, error)
}

// removeLeftovers removes the containers and then the networks that carry
// every given label. Containers go first because Docker refuses to remove a
// network with a container attached. A failure in one half does not skip the
// other, and a resource that is already gone counts as removed, so a teardown
// racing this call cannot fail it.
func removeLeftovers(
	ctx context.Context, engine leftoverEngine, labels map[string]string,
) (removed int, err error) {
	var errs []error

	containers, listErr := engine.ListManagedContainers(ctx, labels)
	if listErr != nil {
		errs = append(errs, fmt.Errorf("list containers: %w", listErr))
	}

	for _, c := range containers {
		if err := engine.RemoveContainer(ctx, c.ID); err != nil && !cerrdefs.IsNotFound(err) {
			errs = append(errs, fmt.Errorf("remove container %s of verification %s: %w", c.ID, c.VerificationID, err))

			continue
		}

		removed++
	}

	networks, listErr := engine.ListManagedNetworks(ctx, labels)
	if listErr != nil {
		errs = append(errs, fmt.Errorf("list networks: %w", listErr))
	}

	for _, n := range networks {
		if err := engine.RemoveNetwork(ctx, n.ID); err != nil && !cerrdefs.IsNotFound(err) {
			errs = append(errs, fmt.Errorf("remove network %s of verification %s: %w", n.ID, n.VerificationID, err))

			continue
		}

		removed++
	}

	return removed, errors.Join(errs...)
}

// prepareAttemptNetwork clears what an earlier attempt of the same
// verification left under these labels, then creates the attempt's network. A
// retry reuses the verification ID and so the network name, which an earlier
// attempt's leftover network would otherwise keep taken.
func prepareAttemptNetwork(
	ctx context.Context, engine attemptNetworkEngine, name string, labels map[string]string,
) (string, error) {
	if _, err := removeLeftovers(ctx, engine, labels); err != nil {
		return "", fmt.Errorf("could not remove the leftovers of an earlier attempt: %w", err)
	}

	networkID, err := engine.CreateNetwork(ctx, name, labels)
	if err != nil {
		return "", fmt.Errorf("create isolated network: %w", err)
	}

	return networkID, nil
}

func agentLabels(agentID string) map[string]string {
	return map[string]string{LabelAgentID: agentID}
}

func attemptLabels(agentID, verificationID string) map[string]string {
	return map[string]string{LabelAgentID: agentID, LabelVerificationID: verificationID}
}
