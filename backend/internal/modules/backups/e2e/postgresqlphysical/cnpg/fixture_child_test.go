package cnpg_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/config"
	containers "databasus-backend/internal/lib/testcontainers"
	"databasus-backend/internal/lib/testwait"
)

const (
	childModeVariable     = "DATABASUS_CNPG_CHILD"
	childManifestVariable = "DATABASUS_CNPG_MANIFEST"
	childIDVariable       = "DATABASUS_CNPG_ID"
	childReleaseVariable  = "DATABASUS_CNPG_RELEASE"
	childHoldLimit        = 10 * time.Minute
)

// The child modes each produce one way a run ends: normally, held until its parent kills it, cancelled
// during startup, or failing on one missing prerequisite.
const (
	childSuccess            = "success"
	childHeld               = "held"
	childCancellation       = "cancellation"
	childLiveCancellation   = "live-cancellation"
	childStalledDiagnostics = "stalled-diagnostics"
	childSupervisor         = "supervisor"
	childUnprivileged       = "unprivileged"
	childUnreachableAddress = "unreachable-address"
	childMissingImage       = "missing-image"
	childDockerUnavailable  = "docker-unavailable"
)

func TestCNPGFixtureChild(t *testing.T) {
	variables, _, err := config.ReadVariables(config.LoadSpec{})
	require.NoError(t, err)
	mode := variables[childModeVariable]
	if mode == "" {
		return
	}

	watchCNPGParent(t)

	if mode == childSupervisor {
		nested := startCNPGChild(t, cnpgChildSpec{Mode: childHeld, FixtureID: variables[childIDVariable]})
		nested.waitUntilReady(t)
		manifest, err := os.ReadFile(nested.ManifestPath)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(variables[childManifestVariable], manifest, 0o600))
		require.NoError(
			t,
			os.WriteFile(
				variables[childManifestVariable]+".pid",
				[]byte(strconv.Itoa(nested.command.Process.Pid)),
				0o600,
			),
		)
		require.NoError(t, os.WriteFile(variables[childManifestVariable]+".ready", nil, 0o600))
		<-t.Context().Done()
		return
	}

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	spec := containers.K3sSpec{
		ID:   variables[childIDVariable],
		Logf: t.Logf,
		ObserveResources: func(resources containers.K3sResources) {
			manifest, err := json.Marshal(resources)
			require.NoError(t, err)
			require.NoError(t, os.WriteFile(variables[childManifestVariable], manifest, 0o600))
			if mode == childStalledDiagnostics && len(resources.ContainerIDs) > 0 {
				stallK3sAPI(t, resources.ContainerIDs[0])
				require.NoError(t, os.WriteFile(variables[childManifestVariable]+".stalled", nil, 0o600))
				cancel()
			}

			if mode == childCancellation && len(resources.ContainerIDs) > 0 {
				cancel()
			}
		},
	}

	switch mode {
	case childUnprivileged:
		spec.ShouldRunUnprivileged = true
	case childUnreachableAddress:
		spec.ShouldUseUnreachableAddress = true
	case childMissingImage:
		spec.ExtraImages = []containers.K3sImage{{
			Name:      "docker.io/databasus/absent-test-image:missing",
			Reference: "docker.io/databasus/absent-test-image:missing@sha256:" + strings.Repeat("0", 64),
		}}
	}

	fixture, err := containers.StartK3s(ctx, spec)
	if err != nil {
		if mode == childCancellation || mode == childStalledDiagnostics {
			require.ErrorIs(t, err, context.Canceled)
		}

		t.Fatalf("intentional fixture %s: %v", mode, err)
	}

	t.Cleanup(func() { require.NoError(t, fixture.Terminate(context.WithoutCancel(t.Context()))) })

	authority, err := containers.NewTestCertificateAuthority()
	require.NoError(t, err)
	containers.StartCNPGCluster(
		t,
		containers.CNPGClusterSpec{K3s: fixture, Authority: authority, Image: containers.CNPGPostgres17Image},
	)
	require.NoError(t, os.WriteFile(variables[childManifestVariable]+".ready", nil, 0o600))

	if mode == childLiveCancellation {
		cancel()
		require.ErrorIs(t, ctx.Err(), context.Canceled)
		t.Fatalf("intentional fixture %s: %v", mode, ctx.Err())
	}

	if mode == childHeld {
		t.Log("intentional fixture held until its parent kills it")
		<-t.Context().Done()
	}

	if releasePath := variables[childReleaseVariable]; releasePath != "" {
		testwait.Within(childHoldLimit).Until(t, "the parent releases the child", func(context.Context) error {
			if _, err := os.Stat(releasePath); err != nil {
				return fmt.Errorf("release: %w", err)
			}

			return nil
		})
	}

	if mode != childSuccess {
		t.Fatal(errors.New("unknown child mode " + mode))
	}
}
