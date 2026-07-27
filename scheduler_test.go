package main

import (
	"testing"
	"time"
)

func TestShouldExecuteOperationOncePerScheduledMinute(t *testing.T) {
	op := ScheduledOperation{Time: "10:00", Action: "on"}
	lastExecutionTime := map[string]time.Time{
		"test": time.Date(2026, 7, 27, 10, 0, 0, 0, time.UTC),
	}

	currentTime := time.Date(2026, 7, 27, 10, 0, 30, 0, time.UTC)
	if shouldExecuteOperation(op, currentTime, lastExecutionTime, "test") {
		t.Fatalf("should not execute again within the same scheduled minute")
	}

	nextDay := time.Date(2026, 7, 28, 10, 0, 0, 0, time.UTC)
	if !shouldExecuteOperation(op, nextDay, lastExecutionTime, "test") {
		t.Fatalf("should execute again on the next matching day")
	}
}

func TestIsDayMatchAllowsDailySchedules(t *testing.T) {
	if !isDayMatch(time.Monday, nil) {
		t.Fatalf("daily schedules should match every weekday")
	}

	if !isDayMatch(time.Sunday, []int{}) {
		t.Fatalf("empty weekday list should mean every day")
	}
}

func TestMultipleScheduledOperationsOnSameDay(t *testing.T) {
	operations := []ScheduledOperation{
		{Time: "09:00", Action: "on"},
		{Time: "18:30", Action: "off"},
	}

	lastExecutionTime := map[string]time.Time{}
	currentTime := time.Date(2026, 7, 27, 9, 0, 0, 0, time.UTC)

	if !shouldExecuteOperation(operations[0], currentTime, lastExecutionTime, "server_2026-07-27_09:00_on_每天") {
		t.Fatalf("first scheduled operation should execute at its own time")
	}

	if shouldExecuteOperation(operations[1], currentTime, lastExecutionTime, "server_2026-07-27_18:30_off_每天") {
		t.Fatalf("second scheduled operation should not execute at a different time")
	}
}
