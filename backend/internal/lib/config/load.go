package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

const databaseDsnVariable = "DATABASE_DSN"

// A process started with `docker exec` does not inherit PID 1's environment, so this file is the
// only way it learns the internal password, which is rotated on every container start. The
// location is memory-backed, so the value dies with the container instead of going stale against
// the next rotation.
const defaultPublishedDatabaseDsnPath = "/dev/shm/databasus-database-dsn"

type LoadSpec struct {
	// The storage probe runs before the embedded database exists, so it starts without a
	// connection string.
	IsDatabaseRequired bool

	// RootDirectory holds .env and the databasus-data directory. Empty means the parent of the
	// backend module found upward from the working directory, or / in the image.
	RootDirectory string

	// PublishedDatabaseDsnPath is empty in production, which means the path container startup
	// writes to.
	PublishedDatabaseDsnPath string
}

// LoadReport tells the caller what Load found, so it can log it once a logger exists.
type LoadReport struct {
	EnvFilePath                string
	IsEnvFileLoaded            bool
	IsPublishedDatabaseDsnUsed bool
	// PublishedDatabaseDsnReadError is set when the published file exists but cannot be read.
	// Startup goes on, because the connection string may come from elsewhere.
	PublishedDatabaseDsnReadError error
}

func Load(spec LoadSpec) (Config, LoadReport, error) {
	var loadedConfig Config

	variables, report, err := ReadVariables(spec)
	if err != nil {
		return Config{}, report, err
	}

	err = env.ParseWithOptions(&loadedConfig, env.Options{Environment: variables})
	if err != nil {
		return Config{}, report, fmt.Errorf("parse settings: %w", err)
	}

	err = validate(loadedConfig, spec)
	if err != nil {
		return Config{}, report, err
	}

	setDataPaths(&loadedConfig, getRootDirectory(spec))

	return loadedConfig, report, nil
}

// ReadVariables merges the setting sources into one map: the process environment first, then .env
// and the published connection string for whatever is still unset. An empty value counts as unset,
// so an empty DATABASE_DSN does not hide the published one.
func ReadVariables(spec LoadSpec) (map[string]string, LoadReport, error) {
	report := LoadReport{EnvFilePath: filepath.Join(getRootDirectory(spec), ".env")}

	variables := map[string]string{}

	for _, variable := range os.Environ() {
		name, value, isPair := strings.Cut(variable, "=")
		if isPair && value != "" {
			variables[name] = value
		}
	}

	fileVariables, isEnvFileLoaded, err := readEnvFile(report.EnvFilePath)
	if err != nil {
		return nil, report, fmt.Errorf("read %s: %w", report.EnvFilePath, err)
	}

	report.IsEnvFileLoaded = isEnvFileLoaded

	for name, value := range fileVariables {
		if _, isSet := variables[name]; !isSet && value != "" {
			variables[name] = value
		}
	}

	if _, isSet := variables[databaseDsnVariable]; !isSet {
		publishedDsn, err := readPublishedDatabaseDsn(getPublishedDatabaseDsnPath(spec))
		if err != nil {
			report.PublishedDatabaseDsnReadError = err
		}

		if publishedDsn != "" {
			variables[databaseDsnVariable] = publishedDsn
			report.IsPublishedDatabaseDsnUsed = true
		}
	}

	return variables, report, nil
}

// The Docker image carries no .env, so every setting there comes from the operator or from a code
// default. A development checkout still reads its own .env.
func readEnvFile(envFilePath string) (map[string]string, bool, error) {
	_, err := os.Stat(envFilePath)
	if errors.Is(err, os.ErrNotExist) {
		return nil, false, nil
	}

	if err != nil {
		return nil, false, err
	}

	fileVariables, err := godotenv.Read(envFilePath)
	if err != nil {
		return nil, false, err
	}

	return fileVariables, true, nil
}

func readPublishedDatabaseDsn(publishedDsnPath string) (string, error) {
	publishedDsn, err := os.ReadFile(publishedDsnPath)
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}

	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(publishedDsn)), nil
}

func setDataPaths(loadedConfig *Config, rootDirectory string) {
	loadedConfig.DataDirectory = filepath.Join(rootDirectory, "databasus-data")
	loadedConfig.DataFolder = filepath.Join(loadedConfig.DataDirectory, "backups")
	loadedConfig.TempFolder = filepath.Join(loadedConfig.DataDirectory, "temp")
	loadedConfig.SecretKeyPath = filepath.Join(loadedConfig.DataDirectory, "secret.key")
	loadedConfig.TelemetryInstancePath = filepath.Join(loadedConfig.DataDirectory, "instance.json")
	loadedConfig.LogFilePath = filepath.Join(loadedConfig.DataDirectory, "databasus.log")
}

func getRootDirectory(spec LoadSpec) string {
	if spec.RootDirectory != "" {
		return spec.RootDirectory
	}

	return filepath.Dir(findBackendRoot())
}

func getPublishedDatabaseDsnPath(spec LoadSpec) string {
	if spec.PublishedDatabaseDsnPath != "" {
		return spec.PublishedDatabaseDsnPath
	}

	return defaultPublishedDatabaseDsnPath
}

// findBackendRoot returns the directory holding go.mod, or the filesystem root when there is none,
// as in the image.
func findBackendRoot() string {
	backendRoot, err := os.Getwd()
	if err != nil {
		return "."
	}

	for {
		if _, err := os.Stat(filepath.Join(backendRoot, "go.mod")); err == nil {
			return backendRoot
		}

		parent := filepath.Dir(backendRoot)
		if parent == backendRoot {
			return backendRoot
		}

		backendRoot = parent
	}
}
