package containers

import (
	"bytes"
	"context"
	"io"
	"strings"
	"time"

	"github.com/moby/moby/api/pkg/stdcopy"
	"github.com/moby/moby/client"
)

const (
	k3sLogTailLines       = "60"
	k3sDiagnosticsTimeout = 10 * time.Second
)

// getDiagnostics describes a failed start: the end of k3s's own log and, when the API answered,
// the pods and the latest events. Each part is bounded so a failure stays readable.
func (fixture *K3s) getDiagnostics(ctx context.Context) string {
	ctx, cancel := context.WithTimeout(ctx, k3sDiagnosticsTimeout)
	defer cancel()

	if fixture.container == nil {
		return "k3s diagnostics: no container was started"
	}

	var report strings.Builder
	report.WriteString("k3s log tail:\n")
	report.WriteString(fixture.getLogTail(ctx))

	for _, arguments := range [][]string{
		{"get", "pods", "--all-namespaces", "-o", "wide"},
		{"get", "events", "--all-namespaces", "--sort-by=.lastTimestamp"},
	} {
		output, err := fixture.Kubectl(ctx, KubectlCommand{Arguments: arguments})
		report.WriteString("\nkubectl " + strings.Join(arguments, " ") + ":\n")
		if err != nil {
			report.WriteString(err.Error())
			continue
		}

		report.WriteString(getBoundedText(output))
	}

	return report.String()
}

func (fixture *K3s) getLogTail(ctx context.Context) string {
	logs, err := fixture.docker.ContainerLogs(
		ctx,
		fixture.container.GetContainerID(),
		client.ContainerLogsOptions{ShowStdout: true, ShowStderr: true, Tail: k3sLogTailLines},
	)
	if err != nil {
		return "unavailable: " + err.Error()
	}
	defer func() { _ = logs.Close() }()

	var output bytes.Buffer
	_, _ = stdcopy.StdCopy(&output, &output, io.LimitReader(logs, 4*k3sDiagnosticsLimit))

	return getBoundedText(output.Bytes())
}
