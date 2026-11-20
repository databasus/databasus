// Package e2e recovers Databasus the way an operator does: it runs the application image, saves a
// Databasus backup and the master key from a source installation, loses that installation, and
// recovers it on new data volumes through mounted startup files and through the file import.
package e2e

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"
)

// candidateImageVariable names an image to test instead of building one from the checkout, as the
// browser suite and the upgrade test accept.
const candidateImageVariable = "DATABASUS_IMAGE"

const imageRemovalTimeout = 2 * time.Minute

// candidateImage is built at most once per test process, because every scenario recovers into the
// same image and a build takes minutes.
//
//nolint:gochecknoglobals // process-level test fixture synchronization: one image for the package's tests
var candidateImage = &imageBuild{}

type imageBuild struct {
	once      sync.Once
	reference string
	isBuilt   bool
	err       error
}

func TestMain(m *testing.M) {
	defer removeBuiltImage()

	m.Run()
}

// removeBuiltImage removes the image the run built; a removal failure fails the run.
func removeBuiltImage() {
	if !candidateImage.isBuilt {
		return
	}

	//nolint:forbidigo // TestMain has no testing.T context; the removal has a bounded lifetime.
	removalCtx, cancelRemoval := context.WithTimeout(context.Background(), imageRemovalTimeout)
	defer cancelRemoval()

	output, err := exec.CommandContext(removalCtx, "docker", "image", "rm", candidateImage.reference).CombinedOutput()
	if err != nil {
		panic(fmt.Sprintf("remove image %s: %v\n%s", candidateImage.reference, err, output))
	}
}

// getCandidateImage returns the image DATABASUS_IMAGE names, or builds one from the checkout under
// a tag no concurrent run uses; TestMain removes a built image.
func getCandidateImage(t *testing.T) string {
	t.Helper()

	candidateImage.once.Do(func() {
		//nolint:forbidigo // the runner names the image, outside any configuration of ours
		if namedImage := os.Getenv(candidateImageVariable); namedImage != "" {
			candidateImage.reference = namedImage

			return
		}

		candidateImage.reference, candidateImage.err = buildCandidateImage(t)
		candidateImage.isBuilt = candidateImage.err == nil
	})

	if candidateImage.err != nil {
		t.Fatalf("build the image from the checkout: %v", candidateImage.err)
	}

	return candidateImage.reference
}

func buildCandidateImage(t *testing.T) (string, error) {
	t.Helper()

	suffix := make([]byte, 6)
	if _, err := rand.Read(suffix); err != nil {
		return "", fmt.Errorf("generate the image tag: %w", err)
	}

	reference := "databasus-recovery-cycle-test:" + hex.EncodeToString(suffix)
	t.Logf("building %s from the checkout", reference)

	output, err := exec.CommandContext(context.WithoutCancel(t.Context()),
		"docker", "build", "--tag", reference, getRepositoryRoot(t)).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("docker build: %w\n%s", err, output)
	}

	return reference, nil
}

func getRepositoryRoot(t *testing.T) string {
	t.Helper()

	_, sourceFile, _, isKnown := runtime.Caller(0)
	if !isKnown {
		t.Fatal("locate the source file of the test")
	}

	return filepath.Join(filepath.Dir(sourceFile), "..", "..", "..", "..", "..")
}
