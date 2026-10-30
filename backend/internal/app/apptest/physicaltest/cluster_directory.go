package physicaltest

import (
	"path/filepath"
	"strconv"

	containers "databasus-backend/internal/lib/testcontainers"
)

const restoreWorkDir = "/restore"

type clusterDirectory struct {
	Image           string
	OutputDirectory string
}

func restoredClusterDir(image string) string {
	if containers.PostgresMajorVersion(image) >= 18 {
		return restoreWorkDir + "/" + strconv.Itoa(containers.PostgresMajorVersion(image)) + "/docker"
	}

	return restoreWorkDir + "/data"
}

func hostVolumeDirForMount(directory clusterDirectory) string {
	image, outDir := directory.Image, directory.OutputDirectory
	if containers.PostgresMajorVersion(image) >= 18 {
		return outDir
	}

	return filepath.Join(outDir, "data")
}

func hostClusterDir(directory clusterDirectory) string {
	image, outDir := directory.Image, directory.OutputDirectory
	if containers.PostgresMajorVersion(image) >= 18 {
		return filepath.Join(outDir, strconv.Itoa(containers.PostgresMajorVersion(image)), "docker")
	}

	return filepath.Join(outDir, "data")
}
