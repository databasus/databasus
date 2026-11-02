// Package toolstub lets a test replace an external client tool, such as pg_dump or mysql, with a
// shell script that records how it was started. A test of code that runs the tool then checks the
// arguments and the input the tool received without a database server.
package toolstub

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"

	"github.com/stretchr/testify/require"
)

const invocationSeparator = "--- toolstub invocation ---"

type Spec struct {
	// IsReadingStdin makes the stub read its whole standard input before Script runs.
	IsReadingStdin bool
	// Script is POSIX shell that runs after the stub recorded its start; it decides what the stub
	// writes and how it exits. An empty Script exits with status 0.
	Script string
}

// Stub is a written stub tool and the files it records into.
type Stub struct {
	argsPath  string
	stdinPath string
}

// Write writes the stub as an executable at path, creating its directories, so a test can place it
// where the code looks for the real tool.
func Write(t *testing.T, path string, spec Spec) *Stub {
	t.Helper()

	if runtime.GOOS == "windows" {
		t.Skip("a stub tool needs a POSIX shell")
	}

	recordDir := t.TempDir()
	stub := &Stub{
		argsPath:  filepath.Join(recordDir, "args"),
		stdinPath: filepath.Join(recordDir, "stdin"),
	}

	script := "#!/bin/sh\n" +
		"echo '" + invocationSeparator + "' >> '" + stub.argsPath + "'\n" +
		"for arg in \"$@\"; do printf '%s\\n' \"$arg\" >> '" + stub.argsPath + "'; done\n"

	if spec.IsReadingStdin {
		script += "cat >> '" + stub.stdinPath + "'\n"
	}

	script += spec.Script + "\n"

	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
	require.NoError(t, WriteExecutable(path, script))

	return stub
}

// WriteExecutable holds syscall.ForkLock while the file is open for writing. A process another
// test forks meanwhile would inherit the open descriptor until its exec, and starting the stub in
// that window fails with "text file busy".
func WriteExecutable(path, script string) error {
	syscall.ForkLock.RLock()
	defer syscall.ForkLock.RUnlock()

	return os.WriteFile(path, []byte(script), 0o700)
}

// GetInvocations returns the arguments of every start of the stub, in the order of the starts.
func (s *Stub) GetInvocations(t *testing.T) [][]string {
	t.Helper()

	recordedArgs, err := os.ReadFile(s.argsPath)
	if os.IsNotExist(err) {
		return nil
	}
	require.NoError(t, err)

	var invocations [][]string

	for line := range strings.SplitSeq(strings.TrimSuffix(string(recordedArgs), "\n"), "\n") {
		if line == invocationSeparator {
			invocations = append(invocations, []string{})

			continue
		}

		invocations[len(invocations)-1] = append(invocations[len(invocations)-1], line)
	}

	return invocations
}

// GetLastArgs returns the arguments of the last start, and fails the test when the stub never ran.
func (s *Stub) GetLastArgs(t *testing.T) []string {
	t.Helper()

	invocations := s.GetInvocations(t)
	require.NotEmpty(t, invocations, "the stub tool never ran")

	return invocations[len(invocations)-1]
}

// GetStdin returns everything the stub read from its standard input over all starts.
func (s *Stub) GetStdin(t *testing.T) []byte {
	t.Helper()

	stdin, err := os.ReadFile(s.stdinPath)
	if os.IsNotExist(err) {
		return nil
	}
	require.NoError(t, err)

	return stdin
}
