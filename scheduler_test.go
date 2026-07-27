package main

import (
	"testing"
	"time"
)

func TestShouldExecuteOperationOncePerMinute(t *testing.T) {
	op := ScheduledOperation{Time: "10:02", Action: "on"}
	lastExecutionTime := map[string]time.Time{
		"test": time.Date(2026, 7, 27, 10, 1, 0, 0, time.UTC),
	}

	currentTime := time.Date(2026, 7, 27, 10, 1, 45, 0, time.UTC)
	if shouldExecuteOperation(op, currentTime, lastExecutionTime, "test") {
		t.Fatalf("should not execute again within the same minute")
	}

	nextMinute := time.Date(2026, 7, 27, 10, 2, 0, 0, time.UTC)
	if !shouldExecuteOperation(op, nextMinute, lastExecutionTime, "test") {
		t.Fatalf("should execute again on the next minute")
	}
}
