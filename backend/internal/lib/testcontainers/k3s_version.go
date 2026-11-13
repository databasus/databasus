package containers

import (
	"context"
	"fmt"
	"strings"
)

func (fixture *K3s) GetK3sVersion(ctx context.Context) (string, error) {
	output, err := fixture.Kubectl(ctx, KubectlCommand{
		Arguments: []string{"get", "nodes", "-o", "jsonpath={.items[0].status.nodeInfo.kubeletVersion}"},
	})
	if err != nil {
		return "", fmt.Errorf("read the k3s version: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}
