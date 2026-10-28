package timelines

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_DecideTimelineCompatibility_WithMatchingAndConflictingIdentity_ReturnsExpectedDecision(t *testing.T) {
	expectedSystemIdentifier := "123"

	tests := []struct {
		name                 string
		liveTimelineID       int
		liveSystemIdentifier string
		expectedTimelineID   int
		decisionKind         TimelineDecisionKind
	}{
		{
			name:                 "matching timeline",
			liveTimelineID:       3,
			liveSystemIdentifier: "123",
			expectedTimelineID:   3,
			decisionKind:         TimelineContinue,
		},
		{
			name:                 "newer timeline",
			liveTimelineID:       4,
			liveSystemIdentifier: "123",
			expectedTimelineID:   3,
			decisionKind:         TimelineFailoverDetected,
		},
		{
			name:                 "older timeline",
			liveTimelineID:       2,
			liveSystemIdentifier: "123",
			expectedTimelineID:   3,
			decisionKind:         TimelineRegression,
		},
		{
			name:                 "different cluster",
			liveTimelineID:       3,
			liveSystemIdentifier: "456",
			expectedTimelineID:   3,
			decisionKind:         TimelineDifferentCluster,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			decision := decideTimelineCompatibility(timelineComparison{
				ExpectedSystemIdentifier: &expectedSystemIdentifier,
				LiveTimelineID:           test.liveTimelineID,
				LiveSystemIdentifier:     test.liveSystemIdentifier,
				ExpectedTimelineID:       test.expectedTimelineID,
			})

			require.Equal(t, test.decisionKind, decision.Kind)
			require.Equal(t, test.expectedTimelineID, decision.ExpectedTimelineID)
			require.Equal(t, test.liveTimelineID, decision.LiveTimelineID)
		})
	}
}
