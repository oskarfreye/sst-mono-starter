package handlers

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRunHatchSendsCrewAlerts(t *testing.T) {
	store := &fakeHatchStore{
		result: HatchRunResult{
			Airlocked: 1,
			Events: []HatchRunEvent{{
				EventID:           "EVENT-001",
				Handle:            "maya",
				SeatID:            "SEAT-002",
				MissionID:         "MISSION-001",
				Mission:           "Ship checkout",
				OccurredAt:        "2026-06-20T12:00:00Z",
				CrewAlertContacts: []string{"cofounder@example.com", "Launch chat"},
			}},
		},
	}
	sender := &recordingCrewAlertSender{}
	handler := NewHatchHandler(HatchHandlerConfig{
		Store:           store,
		CrewAlertSender: sender,
		Clock:           func() time.Time { return mustTime(t, "2026-06-20T12:00:00Z") },
		NewID:           fixedID("EVENT-001"),
	})

	got, err := handler.RunHatch(context.Background())
	if err != nil {
		t.Fatalf("RunHatch returned error: %v", err)
	}

	if len(sender.alerts) != 1 {
		t.Fatalf("alerts = %#v, want one crew alert delivery", sender.alerts)
	}
	alert := sender.alerts[0]
	if alert.EventID != "EVENT-001" || alert.Handle != "maya" || alert.Mission != "Ship checkout" || len(alert.Contacts) != 2 {
		t.Fatalf("alert = %#v, want event payload with contacts", alert)
	}
	if !got.Events[0].CrewAlertSent || got.Events[0].CrewAlertError != "" {
		t.Fatalf("event = %#v, want sent crew alert state", got.Events[0])
	}
}

func TestRunHatchAnnotatesCrewAlertDeliveryFailure(t *testing.T) {
	store := &fakeHatchStore{
		result: HatchRunResult{
			Airlocked: 1,
			Events: []HatchRunEvent{{
				EventID:           "EVENT-001",
				Handle:            "maya",
				SeatID:            "SEAT-002",
				MissionID:         "MISSION-001",
				Mission:           "Ship checkout",
				OccurredAt:        "2026-06-20T12:00:00Z",
				CrewAlertContacts: []string{"cofounder@example.com"},
			}},
		},
	}
	sender := &recordingCrewAlertSender{err: errors.New("webhook down")}
	handler := NewHatchHandler(HatchHandlerConfig{
		Store:           store,
		CrewAlertSender: sender,
	})

	got, err := handler.RunHatch(context.Background())
	if err != nil {
		t.Fatalf("RunHatch returned error: %v", err)
	}

	if got.Events[0].CrewAlertSent || got.Events[0].CrewAlertError != "webhook down" {
		t.Fatalf("event = %#v, want crew alert delivery error", got.Events[0])
	}
}

func TestRunHatchAnnotatesMissingCrewAlertSender(t *testing.T) {
	store := &fakeHatchStore{
		result: HatchRunResult{
			Airlocked: 1,
			Events: []HatchRunEvent{{
				EventID:           "EVENT-001",
				Handle:            "maya",
				SeatID:            "SEAT-002",
				MissionID:         "MISSION-001",
				Mission:           "Ship checkout",
				OccurredAt:        "2026-06-20T12:00:00Z",
				CrewAlertContacts: []string{"cofounder@example.com"},
			}},
		},
	}
	handler := NewHatchHandler(HatchHandlerConfig{Store: store})

	got, err := handler.RunHatch(context.Background())
	if err != nil {
		t.Fatalf("RunHatch returned error: %v", err)
	}

	if got.Events[0].CrewAlertSent || got.Events[0].CrewAlertError != "crew alert webhook is not configured" {
		t.Fatalf("event = %#v, want missing sender annotation", got.Events[0])
	}
}

type fakeHatchStore struct {
	result HatchRunResult
	err    error
}

func (s *fakeHatchStore) RunHatch(context.Context, time.Time, func(string) string) (HatchRunResult, error) {
	return s.result, s.err
}

type recordingCrewAlertSender struct {
	alerts []CrewAlert
	err    error
}

func (s *recordingCrewAlertSender) SendCrewAlert(_ context.Context, alert CrewAlert) error {
	if s.err != nil {
		return s.err
	}
	s.alerts = append(s.alerts, alert)
	return nil
}
