package main

import "testing"

func TestBuildCronSpecForDailySchedule(t *testing.T) {
	spec, err := buildCronSpec(ScheduledOperation{Time: "09:00"})
	if err != nil {
		t.Fatalf("buildCronSpec returned error: %v", err)
	}

	if spec != "0 9 * * *" {
		t.Fatalf("expected daily cron spec to be '0 9 * * *', got %q", spec)
	}
}

func TestBuildCronSpecForSpecificWeekdays(t *testing.T) {
	spec, err := buildCronSpec(ScheduledOperation{Time: "18:30", DaysOfWeek: []int{1, 2, 3, 4, 5}})
	if err != nil {
		t.Fatalf("buildCronSpec returned error: %v", err)
	}

	if spec != "30 18 * * 1,2,3,4,5" {
		t.Fatalf("expected weekday cron spec to be '30 18 * * 1,2,3,4,5', got %q", spec)
	}
}
