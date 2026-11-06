package pgrestore

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"databasus-backend/internal/lib/logging"
)

const MaxFailureDiagnosticBytes = 65536

const (
	diagnosticOmissionMarker = "\n[... diagnostic output truncated ...]\n"
	diagnosticSectionMarker  = "[sensitive diagnostic section and remaining output omitted]"
)

var diagnosticEscapePattern = regexp.MustCompile("\x1b(?:\\[[0-?]*[ -/]*[@-~]|\\][^\x07\x1b]*(?:\x07|\x1b\\\\)|[@-_])")

type FailureDiagnostic struct {
	Output      string
	IsTruncated bool
}

type FailureDiagnosticSpec struct {
	Output      string
	Secrets     []string
	IsTruncated bool
}

func GetFailureDiagnostic(spec FailureDiagnosticSpec) *FailureDiagnostic {
	output := spec.Output

	output = diagnosticEscapePattern.ReplaceAllString(strings.ToValidUTF8(output, "\uFFFD"), "")
	output = strings.ReplaceAll(strings.ReplaceAll(output, "\r\n", "\n"), "\r", "\n")
	output = strings.Map(func(character rune) rune {
		if unicode.IsControl(character) && character != '\n' && character != '\t' {
			return -1
		}
		return character
	}, output)

	for _, secret := range spec.Secrets {
		if secret != "" {
			output = strings.ReplaceAll(output, secret, "[redacted]")
		}
	}

	output = logging.Redact(filterDiagnosticSections(output))
	if strings.TrimSpace(output) == "" {
		return nil
	}

	diagnostic := &FailureDiagnostic{Output: output, IsTruncated: spec.IsTruncated}
	if len(output) <= MaxFailureDiagnosticBytes {
		return diagnostic
	}

	beginningBytes := (MaxFailureDiagnosticBytes - len(diagnosticOmissionMarker)) / 2
	endingStart := len(output) - beginningBytes
	for !utf8.RuneStart(output[beginningBytes]) {
		beginningBytes--
	}

	for !utf8.RuneStart(output[endingStart]) {
		endingStart++
	}

	diagnostic.Output = output[:beginningBytes] + diagnosticOmissionMarker + output[endingStart:]
	diagnostic.IsTruncated = true

	return diagnostic
}

func filterDiagnosticSections(output string) string {
	lines := strings.Split(output, "\n")
	for index, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		if strings.HasPrefix(trimmedLine, "DETAIL:") || strings.HasPrefix(trimmedLine, "CONTEXT:") ||
			strings.HasPrefix(trimmedLine, "Command was:") {
			lines[index] = diagnosticSectionMarker

			return strings.Join(lines[:index+1], "\n")
		}
	}

	return output
}
