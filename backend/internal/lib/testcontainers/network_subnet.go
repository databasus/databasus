package containers

import (
	"fmt"
	// Random test subnets avoid collisions and contain no secrets.
	// nosemgrep: go.lang.security.audit.crypto.math_random.math-random-used
	"math/rand/v2"
	"net/netip"
	"strings"

	"github.com/moby/moby/api/types/network"
)

const subnetAttempts = 8

// WithFreeSubnet calls create with subnets from a range outside Docker's default address pools until
// one does not overlap another network. Tests that overlap each other hold many networks at once, and
// taking them from the default pools would leave none for a verification agent, which cannot choose.
func WithFreeSubnet(create func(ipam *network.IPAM) error) error {
	var err error

	for range subnetAttempts {
		err = create(newRandomSubnetIPAM())
		if err == nil || !strings.Contains(err.Error(), "overlaps") {
			return err
		}
	}

	return fmt.Errorf("no free subnet after %d attempts: %w", subnetAttempts, err)
}

func newRandomSubnetIPAM() *network.IPAM {
	subnet := netip.PrefixFrom(netip.AddrFrom4([4]byte{10, byte(160 + rand.N(32)), byte(rand.N(256)), 0}), 24)

	return &network.IPAM{Driver: "default", Options: nil, Config: []network.IPAMConfig{{Subnet: subnet}}}
}
