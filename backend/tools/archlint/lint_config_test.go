package main

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTargetLayoutPattern(t *testing.T) {
	t.Parallel()

	t.Run("matches every target layout expression in the backend lint configuration", func(t *testing.T) {
		t.Parallel()
		lintConfig, err := os.ReadFile("../../.golangci.yml")
		require.NoError(t, err)

		var targetLayoutExpressions []string

		for line := range strings.Lines(string(lintConfig)) {
			expression, isPathExcept := strings.CutPrefix(strings.TrimSpace(line), "- path-except: ")
			if isPathExcept {
				targetLayoutExpressions = append(
					targetLayoutExpressions,
					strings.TrimPrefix(expression, "&target-layout "),
				)
			}
		}

		require.Len(t, targetLayoutExpressions, 2)

		for _, expression := range targetLayoutExpressions {
			isTargetLayout := expression == targetLayoutPattern.String() ||
				strings.HasPrefix(expression, targetLayoutPattern.String()+"|")
			assert.True(t, isTargetLayout, "expression %q does not start with %q", expression, targetLayoutPattern)
		}
	})
}
