package worker

import (
	"testing"

	"pingme-golang/internal/models"
)

func TestEvaluateStateTransition(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		monitor      models.Monitor
		result       CheckResult
		wantStatus   string
		wantFailures int
		wantEvent    EventType
	}{
		{
			name: "success moves unknown to up",
			monitor: models.Monitor{
				LastStatus:          "unknown",
				ConsecutiveFailures: 0,
			},
			result:       CheckResult{Success: true},
			wantStatus:   "up",
			wantFailures: 0,
			wantEvent:    EventTypeNone,
		},
		{
			name: "failure from unknown stays unknown",
			monitor: models.Monitor{
				LastStatus:          "unknown",
				ConsecutiveFailures: 0,
			},
			result:       CheckResult{Success: false},
			wantStatus:   "unknown",
			wantFailures: 1,
			wantEvent:    EventTypeNone,
		},
		{
			name: "single failure keeps monitor up",
			monitor: models.Monitor{
				LastStatus:          "up",
				ConsecutiveFailures: 0,
			},
			result:       CheckResult{Success: false},
			wantStatus:   "up",
			wantFailures: 1,
			wantEvent:    EventTypeNone,
		},
		{
			name: "third failure moves monitor down",
			monitor: models.Monitor{
				LastStatus:          "up",
				ConsecutiveFailures: 2,
			},
			result:       CheckResult{Success: false},
			wantStatus:   "down",
			wantFailures: 3,
			wantEvent:    EventTypeDown,
		},
		{
			name: "success after down triggers recovery",
			monitor: models.Monitor{
				LastStatus:          "down",
				ConsecutiveFailures: 4,
			},
			result:       CheckResult{Success: true},
			wantStatus:   "up",
			wantFailures: 0,
			wantEvent:    EventTypeRecovered,
		},
		{
			name: "failure while already down does not create a new event",
			monitor: models.Monitor{
				LastStatus:          "down",
				ConsecutiveFailures: 3,
			},
			result:       CheckResult{Success: false},
			wantStatus:   "down",
			wantFailures: 4,
			wantEvent:    EventTypeNone,
		},
		{
			name: "success when already up does not trigger event",
			monitor: models.Monitor{
				LastStatus:          "up",
				ConsecutiveFailures: 0,
			},
			result:       CheckResult{Success: true},
			wantStatus:   "up",
			wantFailures: 0,
			wantEvent:    EventTypeNone,
		},
		{
			name: "success resets failures before reaching threshold",
			monitor: models.Monitor{
				LastStatus:          "up",
				ConsecutiveFailures: 2,
			},
			result:       CheckResult{Success: true},
			wantStatus:   "up",
			wantFailures: 0,
			wantEvent:    EventTypeNone,
		},
		{
			name: "success after single failure resets counter",
			monitor: models.Monitor{
				LastStatus:          "up",
				ConsecutiveFailures: 1,
			},
			result:       CheckResult{Success: true},
			wantStatus:   "up",
			wantFailures: 0,
			wantEvent:    EventTypeNone,
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := evaluateStateTransition(testCase.monitor, testCase.result)

			if got.nextStatus != testCase.wantStatus {
				t.Fatalf("nextStatus = %q, want %q", got.nextStatus, testCase.wantStatus)
			}

			if got.nextFailures != testCase.wantFailures {
				t.Fatalf("nextFailures = %d, want %d", got.nextFailures, testCase.wantFailures)
			}

			if got.eventType != testCase.wantEvent {
				t.Fatalf("eventType = %q, want %q", got.eventType, testCase.wantEvent)
			}
		})
	}
}
