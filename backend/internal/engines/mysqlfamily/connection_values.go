package mysqlfamily

import "strings"

// ConnectionValues are the settings that reach the client tools and the driver as text.
type ConnectionValues struct {
	Host     string
	Username string
	Password string
	Database string
}

// Validate refuses a value the client tools or the driver could read as more than a value. A
// database name that starts with "-" reads as an option to a tool given it as a bare argument, and
// a control character can start a new line of the tools' option file. The tool arguments, the
// option file and the driver guard against both on their own; this check names the setting.
func (v ConnectionValues) Validate() error {
	if strings.HasPrefix(v.Database, "-") {
		return &InvalidSettingError{Setting: "database", Reason: `must not start with "-"`}
	}

	for _, namedValue := range []struct {
		setting string
		value   string
	}{
		{"host", v.Host},
		{"username", v.Username},
		{"password", v.Password},
		{"database", v.Database},
	} {
		if hasControlCharacter(namedValue.value) {
			return &InvalidSettingError{
				Setting: namedValue.setting,
				Reason:  "must not contain a line break or another control character",
			}
		}
	}

	return nil
}
