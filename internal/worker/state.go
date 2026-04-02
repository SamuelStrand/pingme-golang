package worker

import "pingme-golang/internal/models"

type stateTransition struct {
	nextStatus   string
	nextFailures int
	eventType    EventType
}

func evaluateStateTransition(monitor models.Monitor, result CheckResult) stateTransition {
	transition := stateTransition{
		nextStatus:   monitor.LastStatus,
		nextFailures: monitor.ConsecutiveFailures,
		eventType:    EventTypeNone,
	}

	if result.Success {
		transition.nextStatus = "up"
		transition.nextFailures = 0
		if monitor.LastStatus == "down" {
			transition.eventType = EventTypeRecovered
		}
		return transition
	}

	transition.nextFailures = monitor.ConsecutiveFailures + 1
	if transition.nextFailures >= 3 {
		transition.nextStatus = "down"
		if monitor.LastStatus != "down" {
			transition.eventType = EventTypeDown
		}
	}

	return transition
}
