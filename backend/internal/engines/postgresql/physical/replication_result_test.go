package physical

import (
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
)

func TestValidateReplicationResult(t *testing.T) {
	t.Parallel()

	for _, scenario := range []struct {
		name    string
		results []*pgconn.Result
		columns int
		isValid bool
	}{
		{name: "accepts one history row with two fields", results: []*pgconn.Result{{Rows: [][][]byte{{[]byte("00000002.history"), []byte("1\t0/1000000\treason\n")}}}}, columns: 2, isValid: true},
		{name: "accepts identity with a null database field", results: []*pgconn.Result{{Rows: [][][]byte{{[]byte("123"), []byte("2"), []byte("0/1000000"), nil}}}}, columns: 4, isValid: true},
		{name: "rejects the response when results are missing", columns: 2},
		{name: "rejects the response with a null result", results: []*pgconn.Result{nil}, columns: 2},
		{name: "rejects the response when rows are missing", results: []*pgconn.Result{{}}, columns: 2},
		{name: "rejects the response with extra results", results: []*pgconn.Result{{}, {}}, columns: 2},
		{name: "rejects the response with extra rows", results: []*pgconn.Result{{Rows: [][][]byte{{}, {}}}}, columns: 2},
		{name: "rejects the response when fields are missing", results: []*pgconn.Result{{Rows: [][][]byte{{[]byte("name")}}}}, columns: 2},
		{name: "rejects the response with an empty filename", results: []*pgconn.Result{{Rows: [][][]byte{{nil, []byte("history")}}}}, columns: 2},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()

			_, err := getReplicationResultRow(scenario.results, scenario.columns)

			if scenario.isValid {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, ErrInvalidTimelineHistory)
			}
		})
	}
}
