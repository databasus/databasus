package shared

import (
	"errors"
	"net"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

const embeddedPostgresPort = 5437

type EmbeddedTargetSpec struct {
	Host               string
	Port               int
	DatabaseName       string
	IsPhysical         bool
	IsSSHTunnelEnabled bool
	SSHBastionHost     string
}

func ValidateNotEmbeddedTarget(spec EmbeddedTargetSpec) error {
	if spec.IsSSHTunnelEnabled && !isLocalEndpointHost(spec.SSHBastionHost) {
		return nil
	}

	if !isLocalEndpointHost(spec.Host) {
		return nil
	}

	if spec.IsPhysical {
		if spec.Port == embeddedPostgresPort {
			return errors.New("backing up Databasus internal PostgreSQL cluster is not allowed")
		}

		return nil
	}

	if strings.EqualFold(spec.DatabaseName, "databasus") {
		return errors.New(
			"backing up Databasus internal database is not allowed. To backup Databasus itself, see https://databasus.com/faq#backup-databasus",
		)
	}

	return nil
}

func isLocalEndpointHost(host string) bool {
	return slices.ContainsFunc(strings.Split(host, ","), isLocalEndpointHostEntry)
}

func isLocalEndpointHostEntry(host string) bool {
	normalizedHost := strings.TrimSpace(host)
	if normalizedHost == "" {
		return true
	}

	if filepath.IsAbs(normalizedHost) || strings.HasPrefix(normalizedHost, "@") {
		return true
	}

	normalizedHost = strings.TrimSuffix(strings.ToLower(normalizedHost), ".")
	switch normalizedHost {
	case "localhost", "host.docker.internal", "172.17.0.1":
		return true
	}

	addressHost := strings.Trim(normalizedHost, "[]")
	parsedIP := net.ParseIP(addressHost)
	if parsedIP == nil {
		parsedIP, _ = parseIPv4AddressAcceptedByLibc(addressHost)
	}

	return parsedIP != nil && (parsedIP.IsLoopback() || parsedIP.IsUnspecified())
}

func parseIPv4AddressAcceptedByLibc(host string) (net.IP, bool) {
	addressParts := strings.Split(host, ".")
	if len(addressParts) < 1 || len(addressParts) > 4 {
		return nil, false
	}

	partValues := make([]uint64, len(addressParts))
	for partIndex, addressPart := range addressParts {
		partValue, err := strconv.ParseUint(addressPart, 0, 32)
		if err != nil {
			return nil, false
		}

		partValues[partIndex] = partValue
	}

	var addressValue uint64
	switch len(partValues) {
	case 1:
		addressValue = partValues[0]
	case 2:
		if partValues[0] > 0xff || partValues[1] > 0xffffff {
			return nil, false
		}
		addressValue = partValues[0]<<24 | partValues[1]
	case 3:
		if partValues[0] > 0xff || partValues[1] > 0xff || partValues[2] > 0xffff {
			return nil, false
		}
		addressValue = partValues[0]<<24 | partValues[1]<<16 | partValues[2]
	case 4:
		for _, partValue := range partValues {
			if partValue > 0xff {
				return nil, false
			}
		}
		addressValue = partValues[0]<<24 | partValues[1]<<16 | partValues[2]<<8 | partValues[3]
	}

	return net.IPv4(
		byte(addressValue>>24),
		byte(addressValue>>16),
		byte(addressValue>>8),
		byte(addressValue),
	), true
}
