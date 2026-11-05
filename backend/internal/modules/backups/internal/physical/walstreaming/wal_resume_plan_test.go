package walstreaming

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"databasus-backend/internal/engines/postgresql/physical/walmath"
)

func TestResumePathPlan(t *testing.T) {
	for _, testCase := range []struct {
		name         string
		resume, slot walmath.WalSegmentNo
		shouldMove   bool
	}{
		{"moves old queued segments when the resume position precedes the slot", 1, 2, true},
		{"keeps queued segments when resume and slot positions match", 2, 2, false},
		{"keeps queued segments when resume is ahead of the slot", 3, 2, false},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(
				t,
				testCase.shouldMove,
				shouldRealignResumePath(
					resumePathState{ResumeSegmentNo: testCase.resume, SlotSegmentNo: &testCase.slot},
				),
			)
		})
	}
	t.Run("keeps the queue when the source has no available slot position", func(t *testing.T) {
		assert.False(t, shouldRealignResumePath(resumePathState{ResumeSegmentNo: 60}))
	})
}
