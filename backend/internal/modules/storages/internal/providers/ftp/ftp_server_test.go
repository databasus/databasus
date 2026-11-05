package ftp

import (
	"errors"
	"net/textproto"
	"testing"

	"github.com/jlaffaye/ftp"
	"github.com/stretchr/testify/assert"
)

func TestIsFtpFileUnavailable(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		err      error
		isAbsent bool
	}{
		{
			name:     "reports absence when the file is unavailable",
			err:      &textproto.Error{Code: ftp.StatusFileUnavailable, Msg: "No such file or directory"},
			isAbsent: true,
		},
		{
			name:     "reports no absence when the error is transient",
			err:      &textproto.Error{Code: ftp.StatusFileActionIgnored, Msg: "File busy"},
			isAbsent: false,
		},
		{
			name:     "reports no absence when the command is not implemented",
			err:      &textproto.Error{Code: ftp.StatusCommandNotImplemented, Msg: "DELE not supported"},
			isAbsent: false,
		},
		{
			name:     "reports no absence when the connection fails",
			err:      errors.New("connection reset by peer"),
			isAbsent: false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, testCase.isAbsent, isFtpFileUnavailable(testCase.err))
		})
	}
}
