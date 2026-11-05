package gfs

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
