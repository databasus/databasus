package pipeline

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewPostgresqlSizeEstimate(t *testing.T) {
	t.Parallel()

	errSizeQuery := errors.New("size query failed")

	cases := []struct {
		name                 string
		readings             postgresqlSizeReadings
		expectedSizeEstimate SizeEstimate
	}{
		{
			name: "records the database size and the index sizes when both are read",
			readings: postgresqlSizeReadings{
				RawSizeMb:           2048,
				RawSizeErr:          nil,
				LargestIndexSizesMb: []float64{30, 10},
				IndexSizesErr:       nil,
			},
			expectedSizeEstimate: SizeEstimate{RawDbSizeMb: 2048, LargestIndexSizesMb: []float64{30, 10}},
		},
		{
			name: "keeps an empty list when the dumped tables have no index",
			readings: postgresqlSizeReadings{
				RawSizeMb:           2048,
				RawSizeErr:          nil,
				LargestIndexSizesMb: []float64{},
				IndexSizesErr:       nil,
			},
			expectedSizeEstimate: SizeEstimate{RawDbSizeMb: 2048, LargestIndexSizesMb: []float64{}},
		},
		{
			name: "keeps the database size and records no index sizes when only the index query fails",
			readings: postgresqlSizeReadings{
				RawSizeMb:           2048,
				RawSizeErr:          nil,
				LargestIndexSizesMb: nil,
				IndexSizesErr:       errSizeQuery,
			},
			expectedSizeEstimate: SizeEstimate{RawDbSizeMb: 2048, LargestIndexSizesMb: nil},
		},
		{
			name: "records neither size when the database size query fails",
			readings: postgresqlSizeReadings{
				RawSizeMb:           0,
				RawSizeErr:          errSizeQuery,
				LargestIndexSizesMb: []float64{30, 10},
				IndexSizesErr:       nil,
			},
			expectedSizeEstimate: SizeEstimate{RawDbSizeMb: 0, LargestIndexSizesMb: nil},
		},
	}

	for _, estimateCase := range cases {
		t.Run(estimateCase.name, func(t *testing.T) {
			t.Parallel()

			sizeEstimate := newPostgresqlSizeEstimate(estimateCase.readings)

			assert.Equal(t, estimateCase.expectedSizeEstimate, sizeEstimate)
		})
	}
}
