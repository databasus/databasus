package container

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"
)

// fakeDockerHost stores labeled containers and networks and matches a listing
// filter with AND semantics, as Docker does. Every listing, removal and network
// creation lands in one ordered log, so tests can pin the order of operations.
type fakeDockerHost struct {
	containers []fakeResource
	networks   []fakeResource
	log        []string

	listContainersErr error
	listNetworksErr   error
	removeErrs        map[string]error
}

type fakeResource struct {
	id     string
	labels map[string]string
}

func (f *fakeDockerHost) ListManagedContainers(
	_ context.Context, labels map[string]string,
) ([]ManagedResource, error) {
	f.log = append(f.log, "list containers "+formatLabels(labels))
	if f.listContainersErr != nil {
		return nil, f.listContainersErr
	}

	return matchLabels(f.containers, labels), nil
}

func (f *fakeDockerHost) ListManagedNetworks(
	_ context.Context, labels map[string]string,
) ([]ManagedResource, error) {
	f.log = append(f.log, "list networks "+formatLabels(labels))
	if f.listNetworksErr != nil {
		return nil, f.listNetworksErr
	}

	return matchLabels(f.networks, labels), nil
}

func (f *fakeDockerHost) RemoveContainer(_ context.Context, id string) error {
	f.log = append(f.log, "remove container "+id)
	if err := f.removeErrs[id]; err != nil {
		return err
	}

	f.containers = slices.DeleteFunc(f.containers, func(r fakeResource) bool { return r.id == id })

	return nil
}

func (f *fakeDockerHost) RemoveNetwork(_ context.Context, id string) error {
	f.log = append(f.log, "remove network "+id)
	if err := f.removeErrs[id]; err != nil {
		return err
	}

	f.networks = slices.DeleteFunc(f.networks, func(r fakeResource) bool { return r.id == id })

	return nil
}

func (f *fakeDockerHost) CreateNetwork(
	_ context.Context, name string, labels map[string]string,
) (string, error) {
	f.log = append(f.log, "create network "+name)
	id := fmt.Sprintf("net-%d", len(f.networks)+1)
	f.addNetwork(id, labels)

	return id, nil
}

func matchLabels(resources []fakeResource, filter map[string]string) []ManagedResource {
	matched := []ManagedResource{}
	for _, r := range resources {
		hasAll := true
		for key, value := range filter {
			if r.labels[key] != value {
				hasAll = false

				break
			}
		}

		if hasAll {
			matched = append(matched, ManagedResource{ID: r.id, VerificationID: r.labels[LabelVerificationID]})
		}
	}

	return matched
}

// formatLabels renders a label filter in a stable order for the log.
func formatLabels(labels map[string]string) string {
	pairs := []string{}
	for _, key := range slices.Sorted(maps.Keys(labels)) {
		pairs = append(pairs, key+"="+labels[key])
	}

	return strings.Join(pairs, ",")
}

func resourceIDs(resources []fakeResource) []string {
	ids := []string{}
	for _, r := range resources {
		ids = append(ids, r.id)
	}

	return ids
}

func (f *fakeDockerHost) addContainer(id string, labels map[string]string) {
	f.containers = append(f.containers, fakeResource{id: id, labels: labels})
}

func (f *fakeDockerHost) addNetwork(id string, labels map[string]string) {
	f.networks = append(f.networks, fakeResource{id: id, labels: labels})
}

func (f *fakeDockerHost) failRemoval(id string, err error) {
	if f.removeErrs == nil {
		f.removeErrs = map[string]error{}
	}

	f.removeErrs[id] = err
}

func (f *fakeDockerHost) containerIDs() []string {
	return resourceIDs(f.containers)
}

func (f *fakeDockerHost) networkIDs() []string {
	return resourceIDs(f.networks)
}
