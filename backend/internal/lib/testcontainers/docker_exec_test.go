package containers

import (
	"context"
	"fmt"
	"io"
	"testing"

	"github.com/moby/moby/client"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"

	"databasus-backend/internal/lib/testwait"
)

func TestDockerExec(t *testing.T) {
	t.Parallel()

	for _, shouldStreamInput := range []bool{false, true} {
		t.Run(
			fmt.Sprintf("returns cancellation after attachment with streaming input %t", shouldStreamInput),
			func(t *testing.T) {
				t.Parallel()
				container := bootContainer(
					t,
					testcontainers.ContainerRequest{
						Image:      "postgres:17",
						Entrypoint: []string{"sh", "-c"},
						Cmd:        []string{"exec sleep 600"},
					},
				)
				docker, err := client.New(client.FromEnv)
				require.NoError(t, err)
				t.Cleanup(func() { _ = docker.Close() })
				ctx, cancel := context.WithCancel(t.Context())
				defer cancel()
				spec := dockerExec{
					ContainerID: container.GetContainerID(),
					Command:     []string{"sh", "-c", "touch /tmp/attached; exec sleep 600"},
				}

				if shouldStreamInput {
					reader, writer := io.Pipe()
					defer func() { _ = writer.Close() }()
					defer func() { _ = reader.Close() }()
					spec.Stdin = reader
				}

				done := make(chan error, 1)
				go func() { _, err := runDockerExec(ctx, docker, spec); done <- err }()

				testwait.Until(t, "the attached command starts", func(ctx context.Context) error {
					output, err := runDockerExec(
						ctx,
						docker,
						dockerExec{
							ContainerID: container.GetContainerID(),
							Command:     []string{"test", "-f", "/tmp/attached"},
						},
					)
					if err != nil {
						return err
					}

					if output.ExitCode != 0 {
						return fmt.Errorf("command has not attached")
					}

					return nil
				})

				cancel()

				testwait.Until(t, "Docker exec returns after cancellation", func(context.Context) error {
					select {
					case err := <-done:
						require.ErrorIs(t, err, context.Canceled)
						return nil
					default:
						return fmt.Errorf("Docker exec remains blocked after cancellation")
					}
				})
			},
		)
	}
}
