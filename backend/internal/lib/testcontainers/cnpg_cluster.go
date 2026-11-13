package containers

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/testwait"
)

const (
	CNPGBackupRole      = "databasus_backup"
	CNPGBackupRoleGroup = "databasus_backup_group"
	CNPGDatabase        = "testdb"
	cnpgClusterName     = "source"
	cnpgRouteName       = "source-route"
	cnpgStartupLimit    = 5 * time.Minute
)

type CNPGClusterSpec struct {
	K3s       *K3s
	Authority *TestCertificateAuthority
	Image     string
}

// CNPGCluster is one CloudNativePG Cluster with two PostgreSQL nodes in its own namespace. Databasus
// reaches it through Endpoint, a NodePort Kubernetes Service with the selector of CloudNativePG's
// -rw Service, so the operator alone decides which node serves the address.
type CNPGCluster struct {
	K3s       *K3s
	Authority *TestCertificateAuthority
	Namespace string
	Endpoint  Endpoint
	Image     string
	Nodes     [2]string
	// ServerCertificate is the certificate the source presents, which the TLS proxy presents too.
	ServerCertificate ServerCertificate
	// issuer signed ServerCertificate, and verifiedName is a name it holds; the harness's own
	// connections verify against them, so a certificate Databasus must refuse still lets the
	// harness drive the cluster over verified TLS.
	issuer       *TestCertificateAuthority
	verifiedName string
	// killedPods records the pod UID of each node KillPrimary removed.
	killedPods map[string]string
}

func StartCNPGCluster(t *testing.T, spec CNPGClusterSpec) *CNPGCluster {
	t.Helper()

	startedAt := time.Now()
	cluster := &CNPGCluster{
		K3s:        spec.K3s,
		Authority:  spec.Authority,
		Namespace:  "cnpg-" + strings.ReplaceAll(uuid.NewString(), "-", "")[:16],
		Image:      spec.Image,
		Nodes:      [2]string{cnpgClusterName + "-1", cnpgClusterName + "-2"},
		killedPods: map[string]string{},
	}
	t.Cleanup(func() { cluster.delete(t) })

	cluster.kubectl(t, "create", "namespace", cluster.Namespace)
	cluster.applySecrets(t)
	cluster.apply(t, cluster.getClusterManifest(spec.Image))
	cluster.apply(t, cluster.getRouteManifest())

	port, err := strconv.Atoi(strings.TrimSpace(cluster.kubectl(
		t, "get", "service", cnpgRouteName, "-o", "jsonpath={.spec.ports[0].nodePort}",
	)))
	require.NoError(t, err)
	cluster.Endpoint = Endpoint{Host: spec.K3s.Address, Port: port}

	cluster.RequireReady(t)
	t.Logf(
		"CloudNativePG cluster %s ready at %s:%d after %s",
		cluster.Namespace,
		cluster.Endpoint.Host,
		port,
		time.Since(startedAt),
	)

	return cluster
}

// RequireReady waits until the operator reports the cluster healthy, both PostgreSQL nodes run in
// their expected roles and the route serves the current primary.
func (cluster *CNPGCluster) RequireReady(t *testing.T) {
	t.Helper()

	testwait.Within(cnpgStartupLimit).
		Until(t, "CloudNativePG reports a healthy cluster", func(ctx context.Context) error {
			status := cluster.getStatus(ctx, t)
			if status.Phase != "Cluster in healthy state" || status.ReadyInstances != 2 ||
				status.CurrentPrimary == "" || status.CurrentPrimary != status.TargetPrimary {
				return fmt.Errorf("cluster status %+v", status)
			}

			return nil
		})

	identity := cluster.GetIdentity(t)
	for _, node := range cluster.Nodes {
		testwait.Within(cnpgStartupLimit).
			Until(t, "PostgreSQL node has its expected role", func(ctx context.Context) error {
				value, err := cluster.queryNode(ctx, NodeQuery{Node: node, SQL: "SELECT pg_is_in_recovery()"})
				if err != nil {
					return err
				}

				if (value == "t") != (node != identity.Node) {
					return fmt.Errorf("%s in recovery=%s while %s is primary", node, value, identity.Node)
				}

				return nil
			})
	}
}

func (cluster *CNPGCluster) applySecrets(t *testing.T) {
	t.Helper()

	cluster.applyServerCertificate(t, CNPGCertificateTrusted)
	for _, secret := range []map[string]any{
		cluster.getSecret(cnpgSecretSpec{Name: "backup-role", Kind: "kubernetes.io/basic-auth", Values: map[string]string{
			"username": CNPGBackupRole,
			"password": PostgresPassword,
		}}),
		cluster.getSecret(cnpgSecretSpec{Name: "database-owner", Kind: "kubernetes.io/basic-auth", Values: map[string]string{
			"username": PostgresUsername,
			"password": PostgresPassword,
		}}),
	} {
		cluster.apply(t, secret)
	}
}

func (cluster *CNPGCluster) getServiceNames() []string {
	var names []string
	for _, service := range []string{"-rw", "-r", "-ro"} {
		name := cnpgClusterName + service
		names = append(
			names,
			name,
			name+"."+cluster.Namespace,
			name+"."+cluster.Namespace+".svc",
			name+"."+cluster.Namespace+".svc.cluster.local",
		)
	}

	return names
}

type cnpgSecretSpec struct {
	Name   string
	Kind   string
	Values map[string]string
}

// getSecret labels the secret for reloading: CloudNativePG applies a changed user-provided secret, such
// as a rotated server certificate, only when the secret carries cnpg.io/reload.
func (cluster *CNPGCluster) getSecret(spec cnpgSecretSpec) map[string]any {
	return map[string]any{
		"apiVersion": "v1",
		"kind":       "Secret",
		"metadata": map[string]any{
			"name":      spec.Name,
			"namespace": cluster.Namespace,
			"labels":    map[string]string{"cnpg.io/reload": ""},
		},
		"type":       spec.Kind,
		"stringData": spec.Values,
	}
}

// The source is configured only through its manifest, as the user documentation recommends. The
// database owner is the test harness's application role; besides it the manifest declares only the
// backup role and its group, and no network superuser access.
func (cluster *CNPGCluster) getClusterManifest(image string) map[string]any {
	return map[string]any{
		"apiVersion": "postgresql.cnpg.io/v1",
		"kind":       "Cluster",
		"metadata":   map[string]any{"name": cnpgClusterName, "namespace": cluster.Namespace},
		"spec": map[string]any{
			"instances": 2,
			"imageName": image,
			// Databasus's replication connection never leaves on its own, so CloudNativePG's default
			// 180-second smart shutdown would hold every hibernation for its full length before the
			// fast shutdown that ends it anyway.
			"smartShutdownTimeout": 5,
			"storage":              map[string]any{"size": "1Gi"},
			"bootstrap": map[string]any{"initdb": map[string]any{
				"database":      CNPGDatabase,
				"owner":         PostgresUsername,
				"secret":        map[string]any{"name": "database-owner"},
				"dataChecksums": true,
				"postInitSQL": []string{
					"CREATE ROLE " + CNPGBackupRoleGroup + " NOLOGIN",
					"GRANT EXECUTE ON FUNCTION pg_switch_wal() TO " + CNPGBackupRoleGroup,
				},
				"postInitApplicationSQL": []string{
					"GRANT EXECUTE ON FUNCTION pg_switch_wal() TO " + CNPGBackupRoleGroup,
					// The harness drives WAL and checkpoints and reads progress views through the
					// owner; Databasus's backup role gets none of these.
					"GRANT pg_checkpoint, pg_monitor TO " + PostgresUsername,
					"GRANT EXECUTE ON FUNCTION pg_switch_wal() TO " + PostgresUsername,
					"GRANT EXECUTE ON FUNCTION pg_read_file(text) TO " + PostgresUsername,
					"GRANT EXECUTE ON FUNCTION pg_read_binary_file(text) TO " + PostgresUsername,
				},
			}},
			"postgresql": map[string]any{
				"parameters": map[string]string{
					"summarize_wal": "on",
					// CloudNativePG's default of 512MB would hide slot loss within a scenario's WAL.
					"wal_keep_size": "64MB",
					"max_wal_size":  "256MB",
					"min_wal_size":  "32MB",
				},
				"pg_hba": []string{
					"hostnossl all " + CNPGBackupRole + " all reject",
					"hostnossl replication " + CNPGBackupRole + " all reject",
					"hostssl replication " + CNPGBackupRole + " all scram-sha-256",
					"hostssl all " + CNPGBackupRole + " all scram-sha-256",
				},
			},
			"managed": map[string]any{"roles": []map[string]any{{
				"name":           CNPGBackupRole,
				"ensure":         "present",
				"login":          true,
				"replication":    true,
				"inRoles":        []string{CNPGBackupRoleGroup},
				"passwordSecret": map[string]any{"name": "backup-role"},
			}}},
			"certificates": map[string]any{"serverCASecret": "server-ca", "serverTLSSecret": "server-tls"},
			"replicationSlots": map[string]any{
				"highAvailability":    map[string]any{"enabled": true},
				"synchronizeReplicas": map[string]any{"enabled": false},
				"updateInterval":      cnpgSlotUpdateIntervalSeconds,
			},
		},
	}
}

func (cluster *CNPGCluster) getRouteManifest() map[string]any {
	return map[string]any{
		"apiVersion": "v1",
		"kind":       "Service",
		"metadata":   map[string]any{"name": cnpgRouteName, "namespace": cluster.Namespace},
		"spec": map[string]any{
			"type": "NodePort",
			"selector": map[string]string{
				"cnpg.io/cluster":      cnpgClusterName,
				"cnpg.io/instanceRole": "primary",
			},
			"ports": []map[string]any{{"name": "postgres", "port": 5432, "targetPort": 5432, "protocol": "TCP"}},
		},
	}
}

type cnpgStatus struct {
	Phase          string `json:"phase"`
	CurrentPrimary string `json:"currentPrimary"`
	TargetPrimary  string `json:"targetPrimary"`
	ReadyInstances int    `json:"readyInstances"`
}

func (cluster *CNPGCluster) getStatus(ctx context.Context, t *testing.T) cnpgStatus {
	t.Helper()

	output, err := cluster.K3s.Kubectl(ctx, KubectlCommand{Arguments: []string{
		"-n", cluster.Namespace, "get", "cluster", cnpgClusterName, "-o", "jsonpath={.status}",
	}})
	require.NoError(t, err)

	var status cnpgStatus
	require.NoError(t, json.Unmarshal(output, &status))

	return status
}

func (cluster *CNPGCluster) apply(t *testing.T, manifest map[string]any) {
	t.Helper()

	body, err := json.Marshal(manifest)
	require.NoError(t, err)
	require.NoError(t, cluster.K3s.ApplyManifest(t.Context(), string(body)))
}

func (cluster *CNPGCluster) kubectl(t *testing.T, arguments ...string) string {
	t.Helper()

	if arguments[0] != "create" {
		arguments = append([]string{"-n", cluster.Namespace}, arguments...)
	}

	output, err := cluster.K3s.Kubectl(t.Context(), KubectlCommand{Arguments: arguments})
	require.NoError(t, err)

	return string(output)
}

// Deleting the namespace removes the cluster, its volumes and its route; the k3s container stays for
// the next scenario.
func (cluster *CNPGCluster) delete(t *testing.T) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.WithoutCancel(t.Context()), cnpgStartupLimit)
	defer cancel()

	if t.Failed() {
		output, err := cluster.K3s.Kubectl(ctx, KubectlCommand{Arguments: []string{
			"-n", cluster.Namespace, "get", "cluster,pods,events", "-o", "wide",
		}})
		t.Logf("CloudNativePG cluster %s diagnostics: %s %v", cluster.Namespace, getBoundedText(output), err)
	}

	_, err := cluster.K3s.Kubectl(ctx, KubectlCommand{Arguments: []string{
		"delete", "namespace", cluster.Namespace, "--wait=true", "--timeout=" + cnpgStartupLimit.String(),
	}})
	if err != nil {
		t.Errorf("delete CloudNativePG namespace %s: %v", cluster.Namespace, err)
	}
}

// The operator copies user slots to standbys and re-reads them every interval; one second keeps a
// standby's copy close to the primary's position without waiting on time.
const cnpgSlotUpdateIntervalSeconds = 1
