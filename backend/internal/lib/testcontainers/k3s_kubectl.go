package containers

import (
	"context"
	"fmt"
	"io"
	"strings"
)

type KubectlCommand struct {
	Arguments []string
	Stdin     io.ReadCloser
}

// Kubectl runs kubectl inside the k3s container, which reads the cluster's own kubeconfig there.
func (fixture *K3s) Kubectl(ctx context.Context, command KubectlCommand) ([]byte, error) {
	output, err := fixture.exec(
		ctx,
		dockerExec{Command: append([]string{"kubectl"}, command.Arguments...), Stdin: command.Stdin},
	)
	if err != nil {
		return nil, err
	}

	if output.ExitCode != 0 {
		return output.Stdout, fmt.Errorf(
			"%w: kubectl %s exited with %d: %s",
			ErrKubernetesCommandFailed,
			strings.Join(command.Arguments, " "),
			output.ExitCode,
			getBoundedText(output.Stderr),
		)
	}

	return output.Stdout, nil
}

// ApplyManifest applies a manifest server-side, so a repeated apply of the same objects is a no-op.
func (fixture *K3s) ApplyManifest(ctx context.Context, manifest string) error {
	_, err := fixture.Kubectl(ctx, KubectlCommand{
		Arguments: []string{"apply", "--server-side", "--force-conflicts", "-f", "-"},
		Stdin:     io.NopCloser(strings.NewReader(manifest)),
	})

	return err
}

func (fixture *K3s) exec(ctx context.Context, command dockerExec) (dockerExecOutput, error) {
	command.ContainerID = fixture.container.GetContainerID()

	return runDockerExec(ctx, fixture.docker.Client, command)
}

func getBoundedText(output []byte) string {
	if len(output) > k3sDiagnosticsLimit {
		output = output[len(output)-k3sDiagnosticsLimit:]
	}

	return strings.ReplaceAll(strings.TrimSpace(string(output)), PostgresPassword, "[REDACTED]")
}
