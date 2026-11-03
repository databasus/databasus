package upgrade

import (
	"context"
	"errors"
	"fmt"

	cerrdefs "github.com/containerd/errdefs"
	"github.com/moby/moby/api/types/mount"
	"github.com/moby/moby/client"
)

func (d *dockerHost) removeAgentContainer(ctx context.Context, containerID string) error {
	inspection, err := d.client.ContainerInspect(ctx, containerID, client.ContainerInspectOptions{})
	if cerrdefs.IsNotFound(err) {
		return nil
	}

	if err != nil {
		return err
	}

	_, err = d.client.ContainerRemove(ctx, containerID, client.ContainerRemoveOptions{Force: true, RemoveVolumes: true})
	if err != nil {
		return err
	}

	var cleanupErrors []error
	for _, volumeMount := range inspection.Container.Mounts {
		if volumeMount.Type != mount.TypeVolume {
			continue
		}

		_, err := d.client.VolumeInspect(ctx, volumeMount.Name, client.VolumeInspectOptions{})
		if cerrdefs.IsNotFound(err) {
			continue
		}

		if err != nil {
			cleanupErrors = append(cleanupErrors, fmt.Errorf("inspect removed agent volume: %w", err))
			continue
		}

		cleanupErrors = append(
			cleanupErrors,
			fmt.Errorf("agent volume %s remains after container removal", volumeMount.Name),
		)
		_, removeErr := d.client.VolumeRemove(ctx, volumeMount.Name, client.VolumeRemoveOptions{Force: true})
		cleanupErrors = append(cleanupErrors, removeErr)
	}

	return errors.Join(cleanupErrors...)
}
