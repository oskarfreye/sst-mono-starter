package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type crewAlertSender interface {
	SendCrewAlert(ctx context.Context, alert CrewAlert) error
}

type CrewAlert struct {
	EventID    string   `json:"event_id"`
	Handle     string   `json:"handle"`
	SeatID     string   `json:"seat_id"`
	MissionID  string   `json:"mission_id"`
	Mission    string   `json:"mission"`
	OccurredAt string   `json:"occurred_at"`
	Contacts   []string `json:"contacts"`
}

type webhookCrewAlertSender struct {
	url    string
	client *http.Client
}

func newWebhookCrewAlertSender(url string) *webhookCrewAlertSender {
	return &webhookCrewAlertSender{
		url: url,
		client: &http.Client{
			Timeout: 8 * time.Second,
		},
	}
}

func (s *webhookCrewAlertSender) SendCrewAlert(ctx context.Context, alert CrewAlert) error {
	body, err := json.Marshal(alert)
	if err != nil {
		return fmt.Errorf("encode crew alert: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build crew alert request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "the-airlock-hatch/1.0")

	res, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("send crew alert: %w", err)
	}
	defer res.Body.Close()
	_, _ = io.Copy(io.Discard, res.Body)
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf("crew alert webhook status %d", res.StatusCode)
	}
	return nil
}
