package containers

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/testwait"
)

// Hibernate stops every PostgreSQL node of the cluster declaratively and keeps its volumes, so a
// recovery can prove it needs nothing from the source. The route must then refuse connections.
func (cluster *CNPGCluster) Hibernate(t *testing.T) {
	t.Helper()

	cluster.kubectl(t, "annotate", "cluster", cnpgClusterName, "cnpg.io/hibernation=on", "--overwrite")

	testwait.Until(t, "CloudNativePG hibernates the cluster", func(ctx context.Context) error {
		output, err := cluster.K3s.Kubectl(ctx, KubectlCommand{Arguments: []string{
			"-n", cluster.Namespace, "get", "pods", "-l", "cnpg.io/cluster=" + cnpgClusterName, "-o", "name",
		}})
		if err != nil {
			return err
		}

		if pods := strings.TrimSpace(string(output)); pods != "" {
			return fmt.Errorf("pods still run: %s", pods)
		}

		condition, err := cluster.K3s.Kubectl(ctx, KubectlCommand{Arguments: []string{
			"-n", cluster.Namespace, "get", "cluster", cnpgClusterName,
			"-o", `jsonpath={.status.conditions[?(@.type=="cnpg.io/hibernation")].status}`,
		}})
		if err != nil {
			return err
		}

		if strings.TrimSpace(string(condition)) != "True" {
			return fmt.Errorf("hibernation condition is %q", condition)
		}

		return nil
	})

	dialer := net.Dialer{Timeout: 3 * time.Second}
	connection, err := dialer.DialContext(
		t.Context(),
		"tcp",
		net.JoinHostPort(cluster.Endpoint.Host, strconv.Itoa(cluster.Endpoint.Port)),
	)
	if err == nil {
		_ = connection.Close()
	}

	require.Error(t, err, "the route of a hibernated cluster must refuse connections")
}
