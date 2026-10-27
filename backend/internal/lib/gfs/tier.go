package gfs

import "strings"

// Tier names the grandfather-father-son level that retained an item. A single item can satisfy
// several levels at once, which is why retention returns every tier that claimed it rather than one.
type Tier string

const (
	TierHourly  Tier = "hourly"
	TierDaily   Tier = "daily"
	TierWeekly  Tier = "weekly"
	TierMonthly Tier = "monthly"
	TierYearly  Tier = "yearly"
)

// An item can satisfy several levels at once, so a retention log line has to name all of them.
func FormatTiers(tiers []Tier) string {
	names := make([]string, len(tiers))
	for i, tier := range tiers {
		names[i] = string(tier)
	}

	return strings.Join(names, "+")
}
