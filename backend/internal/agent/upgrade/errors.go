package upgrade

import "errors"

var ErrUpgradeRestart = errors.New("agent upgraded, restart required")
