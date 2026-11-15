package upgrade

import (
	"context"

	"github.com/moby/moby/client"

	containers "databasus-backend/internal/lib/testcontainers"
)

// dockerHost adds the upgrade flow's own resources to the shared Docker host. The client and the
// labels serve the agent and retirement fixtures, which create containers the shared host does not
// describe.
type dockerHost struct {
	*containers.DockerHost

	client *client.Client
	labels map[string]string
}

func newDockerHost() (*dockerHost, error) {
	sharedHost, err := containers.NewDockerHost()
	if err != nil {
		return nil, err
	}

	return &dockerHost{
		DockerHost: sharedHost,
		client:     sharedHost.GetClient(),
		labels:     sharedHost.GetLabels(),
	}, nil
}

// removeResources removes everything the flow may have created, skipping what does not exist.
func (d *dockerHost) removeResources(ctx context.Context, resources upgradeResources) error {
	return d.RemoveResources(ctx, resources.getDockerResources())
}

// getRemainingResources names every resource of the flow that Docker still has.
func (d *dockerHost) getRemainingResources(ctx context.Context, resources upgradeResources) ([]string, error) {
	return d.GetRemainingResources(ctx, resources.getDockerResources())
}
