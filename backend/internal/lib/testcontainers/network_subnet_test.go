package containers

import (
	"errors"
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
