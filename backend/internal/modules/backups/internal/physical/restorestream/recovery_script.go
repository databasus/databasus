package restorestream

import _ "embed"

//go:embed recovery_script.sh
var recoveryScript string

func GetRecoveryScript() string { return recoveryScript }
