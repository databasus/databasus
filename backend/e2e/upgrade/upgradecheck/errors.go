package upgradecheck

import "errors"

var ErrItemsDiffer = errors.New("items differ after the upgrade")
