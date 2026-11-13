package containers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	CNPGOperatorVersion   = "1.30.1"
	cnpgManifestFile      = "cnpg-1.30.1.yaml"
	cnpgManifestSHA256    = "37237f145d8138256ea25ae830f87759255665ff08f8d552fdd8224a5ec032fb"
	cnpgOperatorNamespace = "cnpg-system"
	cnpgOperatorName      = "cnpg-controller-manager"
)

// The release manifest pulls the operator image on every start. k3s has no registry to pull from, so
// the one pull policy line is switched to the imported image.
const (
	cnpgReleasePullPolicy  = "imagePullPolicy: Always"
	cnpgImportedPullPolicy = "imagePullPolicy: IfNotPresent"
)

func (fixture *K3s) installCNPGOperator(ctx context.Context) error {
	manifest, err := readCNPGManifest()
	if err != nil {
		return err
	}

	if err := fixture.ApplyManifest(ctx, manifest); err != nil {
		return fmt.Errorf("apply the CloudNativePG operator: %w", err)
	}

	for _, arguments := range [][]string{
		{"wait", "--for=condition=Established", "--timeout=" + k3sStartupLimit.String(), "crd/clusters.postgresql.cnpg.io"},
		{"-n", cnpgOperatorNamespace, "rollout", "status", "deployment/" + cnpgOperatorName, "--timeout=" + k3sStartupLimit.String()},
	} {
		if _, err := fixture.Kubectl(ctx, KubectlCommand{Arguments: arguments}); err != nil {
			return fmt.Errorf("wait for the CloudNativePG operator: %w", err)
		}
	}

	version, err := fixture.GetOperatorVersion(ctx)
	if err != nil {
		return err
	}

	fixture.logf("k3s fixture %s CloudNativePG operator: %s", fixture.ID, version)

	return nil
}

func (fixture *K3s) GetOperatorVersion(ctx context.Context) (string, error) {
	output, err := fixture.Kubectl(ctx, KubectlCommand{
		Arguments: []string{
			"-n",
			cnpgOperatorNamespace,
			"exec",
			"deployment/" + cnpgOperatorName,
			"--",
			"/manager",
			"version",
		},
	})
	if err != nil {
		return "", fmt.Errorf("read the CloudNativePG operator version: %w", err)
	}

	version := strings.TrimSpace(string(output))
	if !strings.Contains(version, "Version:"+CNPGOperatorVersion) {
		return "", fmt.Errorf(
			"the running CloudNativePG operator reports %q, expected %s",
			version,
			CNPGOperatorVersion,
		)
	}

	return version, nil
}

func readCNPGManifest() (string, error) {
	_, sourceFile, _, isResolved := runtime.Caller(0)
	if !isResolved {
		return "", fmt.Errorf("resolve the CloudNativePG fixture directory")
	}

	body, err := os.ReadFile(filepath.Join(filepath.Dir(sourceFile), "testdata", "cnpg", cnpgManifestFile))
	if err != nil {
		return "", fmt.Errorf("read the CloudNativePG manifest: %w", err)
	}

	digest := sha256.Sum256(body)
	if hex.EncodeToString(digest[:]) != cnpgManifestSHA256 {
		return "", fmt.Errorf("%w: %s", ErrCNPGManifestChanged, cnpgManifestFile)
	}

	manifest := string(body)
	if strings.Count(manifest, cnpgReleasePullPolicy) != 1 {
		return "", fmt.Errorf("%w: expected one %q", ErrCNPGManifestChanged, cnpgReleasePullPolicy)
	}

	return strings.Replace(manifest, cnpgReleasePullPolicy, cnpgImportedPullPolicy, 1), nil
}
