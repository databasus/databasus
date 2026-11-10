package testimages

import (
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

//go:embed images.json
var imageCatalogJSON []byte

type Selection struct {
	Image        string
	Architecture string
}

type imageRecord struct {
	Reference string            `json:"reference"`
	Platforms map[string]string `json:"platforms"`
}

type catalog struct {
	Schema int                    `json:"schema"`
	Images map[string]imageRecord `json:"images"`
}

func GetReference(selection Selection) (string, error) {
	var images catalog
	if err := json.Unmarshal(imageCatalogJSON, &images); err != nil || images.Schema != 1 {
		return "", ErrInvalidCatalog
	}

	image, hasImage := images.Images[selection.Image]
	if !hasImage {
		return "", fmt.Errorf("%w: %s", ErrUnknownImage, selection.Image)
	}

	platform := ""
	switch selection.Architecture {
	case "amd64", "x86_64", "x64":
		platform = "linux-x64"
	case "arm64", "aarch64":
		platform = "linux-arm64"
	}

	digest, hasPlatform := image.Platforms[platform]
	if !hasPlatform {
		return "", fmt.Errorf("%w: %s on %s", ErrUnavailablePlatform, selection.Image, selection.Architecture)
	}

	repository, indexDigest, hasDigest := strings.Cut(image.Reference, "@")
	if !hasDigest || repository == "" || !isDigest(indexDigest) || !isDigest(digest) {
		return "", ErrInvalidCatalog
	}

	return repository + "@" + digest, nil
}

func isDigest(digest string) bool {
	if !strings.HasPrefix(digest, "sha256:") || digest != strings.ToLower(digest) {
		return false
	}

	decodedDigest, err := hex.DecodeString(strings.TrimPrefix(digest, "sha256:"))

	return err == nil && len(decodedDigest) == 32
}
