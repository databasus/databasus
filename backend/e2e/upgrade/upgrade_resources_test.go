package upgrade

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// upgradeResources are the unique names of everything one upgrade flow creates in Docker.
type upgradeResources struct {
	networkName           string
	volumeName            string
	sourceContainerName   string
	previousContainerName string
	currentContainerName  string
}

func newUpgradeResources() (upgradeResources, error) {
	suffix := make([]byte, 6)

	_, err := rand.Read(suffix)
	if err != nil {
		return upgradeResources{}, fmt.Errorf("generate resource names: %w", err)
	}

	prefix := "databasus-upgrade-" + hex.EncodeToString(suffix)

	return upgradeResources{
		networkName:           prefix + "-network",
		volumeName:            prefix + "-data",
		sourceContainerName:   prefix + "-source",
		previousContainerName: prefix + "-previous",
		currentContainerName:  prefix + "-current",
	}, nil
}

func (r upgradeResources) getContainerNames() []string {
	return []string{r.sourceContainerName, r.previousContainerName, r.currentContainerName}
}
