package containers

import (
	"errors"
	"net/netip"
	"testing"

	"github.com/moby/moby/api/types/network"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errPoolOverlaps = errors.New("pool overlaps with other one on this address space")

func TestWithFreeSubnet(t *testing.T) {
	t.Parallel()

	t.Run("takes the subnet from the range outside the default pools", func(t *testing.T) {
		t.Parallel()

		err := WithFreeSubnet(func(ipam *network.IPAM) error {
			require.Len(t, ipam.Config, 1)
			assert.Equal(t, 24, ipam.Config[0].Subnet.Bits())
			assert.GreaterOrEqual(t, ipam.Config[0].Subnet.Addr().As4()[1], byte(160))
			assert.Less(t, ipam.Config[0].Subnet.Addr().As4()[1], byte(192))

			return nil
		})

		require.NoError(t, err)
	})

	t.Run("tries another subnet when one overlaps", func(t *testing.T) {
		t.Parallel()
		attempts := 0

		err := WithFreeSubnet(func(*network.IPAM) error {
			attempts++
			if attempts < 3 {
				return errPoolOverlaps
			}

			return nil
		})

		require.NoError(t, err)
		assert.Equal(t, 3, attempts)
	})

	t.Run("returns another error without trying again", func(t *testing.T) {
		t.Parallel()
		attempts := 0
		errDaemon := errors.New("daemon is gone")

		err := WithFreeSubnet(func(*network.IPAM) error {
			attempts++

			return errDaemon
		})

		require.ErrorIs(t, err, errDaemon)
		assert.Equal(t, 1, attempts)
	})

	t.Run("returns the overlap when every attempt overlaps", func(t *testing.T) {
		t.Parallel()

		err := WithFreeSubnet(func(*network.IPAM) error { return errPoolOverlaps })

		require.ErrorIs(t, err, errPoolOverlaps)
	})
}

func TestIsTestNetworkSubnet(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		subnet        string
		isTestNetwork bool
	}{
		{name: "accepts a subnet at the start of the range", subnet: "10.160.0.0/24", isTestNetwork: true},
		{name: "accepts a subnet at the end of the range", subnet: "10.191.255.0/24", isTestNetwork: true},
		{name: "rejects a subnet of the default pools", subnet: "172.18.0.0/16", isTestNetwork: false},
		{name: "rejects a subnet just past the range", subnet: "10.192.0.0/24", isTestNetwork: false},
		{name: "rejects a subnet wider than the range", subnet: "10.128.0.0/9", isTestNetwork: false},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, testCase.isTestNetwork, IsTestNetworkSubnet(netip.MustParsePrefix(testCase.subnet)))
		})
	}
}
