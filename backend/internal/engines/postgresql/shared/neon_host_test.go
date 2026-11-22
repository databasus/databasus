package shared

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsNeonHost(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		host       string
		isNeonHost bool
	}{
		{name: "accepts an endpoint host", host: "ep-young-dew-123456.c-8.us-east-2.aws.neon.tech", isNeonHost: true},
		{
			name:       "accepts a pooler host",
			host:       "ep-young-dew-123456-pooler.c-8.us-east-2.aws.neon.tech",
			isNeonHost: true,
		},
		{
			name:       "accepts a host with capitals and a trailing dot",
			host:       "EP-YOUNG-DEW-123456.US-EAST-2.AWS.NEON.TECH.",
			isNeonHost: true,
		},
		{
			name:       "accepts a host with surrounding whitespace",
			host:       "  ep-young-dew-123456.neon.tech ",
			isNeonHost: true,
		},
		{name: "accepts the bare domain", host: "neon.tech", isNeonHost: true},
		{name: "rejects a host that only contains the domain", host: "neon.tech.example.com", isNeonHost: false},
		{name: "rejects a host that only ends with the domain letters", host: "notneon.tech", isNeonHost: false},
		{name: "rejects an empty host", host: "", isNeonHost: false},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, testCase.isNeonHost, IsNeonHost(testCase.host))
		})
	}
}
