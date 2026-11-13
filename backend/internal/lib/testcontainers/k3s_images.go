package containers

import (
	"context"
	"fmt"
	"runtime"
	"strings"

	"github.com/moby/moby/client"

	"databasus-backend/internal/lib/testimages"
)

// K3sImage is an image imported into k3s. Name is what Kubernetes manifests refer to, and Reference
// is the pinned platform image the host's Docker provides.
type K3sImage struct {
	Name      string
	Reference string
}

const (
	CNPGPostgres17Image = "ghcr.io/cloudnative-pg/postgresql:17.11-202610260819-standard-trixie"
	CNPGPostgres18Image = "ghcr.io/cloudnative-pg/postgresql:18.6-202610260820-standard-trixie"
	cnpgOperatorImage   = "ghcr.io/cloudnative-pg/cloudnative-pg:1.30.1"
)

//nolint:gochecknoglobals // Immutable shared table of the images every k3s container imports.
var cnpgImportedImages = []string{
	"rancher/mirrored-pause:3.10.2",
	"rancher/mirrored-coredns-coredns:1.14.7",
	"rancher/local-path-provisioner:v0.0.37",
	"rancher/mirrored-library-busybox:1.37.0",
	cnpgOperatorImage,
	CNPGPostgres17Image,
	CNPGPostgres18Image,
}

func getCNPGImages() ([]K3sImage, error) {
	images := make([]K3sImage, 0, len(cnpgImportedImages))
	for _, image := range cnpgImportedImages {
		reference, err := getCatalogReference(image)
		if err != nil {
			return nil, err
		}

		name, _, _ := strings.Cut(reference, "@")
		images = append(images, K3sImage{Name: name, Reference: reference})
	}

	return images, nil
}

func getCatalogReference(image string) (string, error) {
	reference, err := testimages.GetReference(testimages.Selection{Image: image, Architecture: runtime.GOARCH})
	if err != nil {
		return "", fmt.Errorf("%w: %s: %w", ErrImageUnavailable, image, err)
	}

	return reference, nil
}

func (fixture *K3s) pullImage(ctx context.Context, reference string) error {
	if _, err := fixture.docker.ImageInspect(ctx, reference); err == nil {
		return nil
	}

	response, err := fixture.docker.ImagePull(ctx, reference, client.ImagePullOptions{})
	if err != nil {
		return fmt.Errorf("%w: %s: %w", ErrImageUnavailable, reference, err)
	}
	defer func() { _ = response.Close() }()

	if err := response.Wait(ctx); err != nil {
		return fmt.Errorf("%w: %s: %w", ErrImageUnavailable, reference, err)
	}

	return nil
}

// importImages streams each image from the host's Docker into k3s's containerd and names it as the
// manifests do. No archive touches the host filesystem, so a killed run leaves none behind.
func (fixture *K3s) importImages(ctx context.Context, images []K3sImage) error {
	for _, image := range images {
		if err := fixture.pullImage(ctx, image.Reference); err != nil {
			return err
		}

		if err := fixture.importImage(ctx, image); err != nil {
			return err
		}
	}

	return nil
}

func (fixture *K3s) importImage(ctx context.Context, image K3sImage) error {
	repository, digest, hasDigest := strings.Cut(image.Reference, "@")
	if !hasDigest {
		return fmt.Errorf("%w: %s has no digest", ErrImageUnavailable, image.Reference)
	}

	if index := strings.LastIndex(repository, ":"); index > strings.LastIndex(repository, "/") {
		repository = repository[:index]
	}

	archive, err := fixture.docker.ImageSave(ctx, []string{image.Reference})
	if err != nil {
		return fmt.Errorf("%w: save %s: %w", ErrImageUnavailable, image.Reference, err)
	}
	defer func() { _ = archive.Close() }()

	for _, command := range []dockerExec{
		{
			Command: []string{"ctr", "-n", "k8s.io", "images", "import", "--base-name", repository, "--digests", "-"},
			Stdin:   archive,
		},
		{Command: []string{"ctr", "-n", "k8s.io", "images", "tag", "--force", repository + "@" + digest, image.Name}},
	} {
		output, err := fixture.exec(ctx, command)
		if err != nil {
			return fmt.Errorf("import %s: %w: %s", image.Reference, err, getBoundedText(output.Stderr))
		}

		if output.ExitCode != 0 {
			return fmt.Errorf(
				"import %s: %q exited with %d: %s",
				image.Reference,
				command.Command,
				output.ExitCode,
				getBoundedText(output.Stderr),
			)
		}
	}

	return nil
}
