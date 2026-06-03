package handlers

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"
)

const stateCacheControl = "public, max-age=10, s-maxage=10, stale-while-revalidate=30"
const eventStreamWindow = 55 * time.Second
const eventStreamInterval = 3 * time.Second

type StateHandlerConfig struct {
	TableName    string
	AWSRegion    string
	LaunchUserID string
	Clock        func() time.Time
}

type StateHandler struct {
	store stateStore
	clock func() time.Time
}

func NewStateHandler(cfg StateHandlerConfig) *StateHandler {
	clock := cfg.Clock
	if clock == nil {
		clock = time.Now
	}

	var store stateStore = launchStateStore{}
	if cfg.TableName != "" {
		store = newDynamoStateStore(cfg.TableName, cfg.AWSRegion, cfg.LaunchUserID)
	}

	return &StateHandler{store: store, clock: clock}
}

func (h *StateHandler) GetState(c *fiber.Ctx) error {
	c.Set(fiber.HeaderCacheControl, stateCacheControl)

	snapshot, err := h.loadSnapshot(c.UserContext())
	if err != nil {
		log.Printf("state: load failed: %v", err)
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "state unavailable"})
	}

	c.Set("X-Airlock-State-Source", snapshot.Source)
	return c.JSON(snapshot)
}

func (h *StateHandler) GetSeats(c *fiber.Ctx) error {
	c.Set(fiber.HeaderCacheControl, stateCacheControl)

	snapshot, err := h.loadSnapshot(c.UserContext())
	if err != nil {
		log.Printf("seats: state load failed: %v", err)
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "seats unavailable"})
	}

	c.Set("X-Airlock-State-Source", snapshot.Source)
	return c.JSON(fiber.Map{
		"snapshot":       snapshot.Snapshot,
		"source":         snapshot.Source,
		"seats":          snapshot.Seats,
		"active_seats":   snapshot.ActiveSeats,
		"vacated_seats":  snapshot.VacatedSeats,
		"cohort_summary": snapshot.CohortSummary,
		"next_seat":      snapshot.NextSeat,
	})
}

func (h *StateHandler) loadSnapshot(ctx context.Context) (stateSnapshot, error) {
	records, err := h.store.Load(ctx)
	if err != nil {
		return stateSnapshot{}, err
	}
	return buildStateSnapshot(records, h.clock()), nil
}

func (h *StateHandler) StreamEvents(c *fiber.Ctx) error {
	c.Set(fiber.HeaderContentType, "text/event-stream; charset=utf-8")
	c.Set(fiber.HeaderCacheControl, "no-store")
	c.Set(fiber.HeaderConnection, "keep-alive")
	c.Set("X-Accel-Buffering", "no")

	ctx := c.UserContext()
	once := c.Query("once") == "1"
	c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
		if _, err := fmt.Fprint(w, "retry: 30000\n\n"); err != nil {
			return
		}
		if err := h.writeStateEvent(ctx, w); err != nil {
			writeSSEError(w, err)
			return
		}
		if err := w.Flush(); err != nil || once {
			return
		}

		ticker := time.NewTicker(eventStreamInterval)
		defer ticker.Stop()
		deadline := time.NewTimer(eventStreamWindow)
		defer deadline.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-deadline.C:
				return
			case <-ticker.C:
				if err := h.writeStateEvent(ctx, w); err != nil {
					writeSSEError(w, err)
					return
				}
				if err := w.Flush(); err != nil {
					return
				}
			}
		}
	})
	return nil
}

func (h *StateHandler) StreamEventsHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("X-Accel-Buffering", "no")

	ctx := r.Context()
	once := r.URL.Query().Get("once") == "1"
	writer := bufio.NewWriter(w)
	if _, err := fmt.Fprint(writer, "retry: 30000\n\n"); err != nil {
		return
	}
	if err := h.writeStateEvent(ctx, writer); err != nil {
		writeSSEError(writer, err)
		return
	}
	if err := writer.Flush(); err != nil || once {
		return
	}

	ticker := time.NewTicker(eventStreamInterval)
	defer ticker.Stop()
	deadline := time.NewTimer(eventStreamWindow)
	defer deadline.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-deadline.C:
			return
		case <-ticker.C:
			if err := h.writeStateEvent(ctx, writer); err != nil {
				writeSSEError(writer, err)
				return
			}
			if err := writer.Flush(); err != nil {
				return
			}
		}
	}
}

func (h *StateHandler) writeStateEvent(ctx context.Context, w *bufio.Writer) error {
	records, err := h.store.Load(ctx)
	if err != nil {
		log.Printf("events: state load failed: %v", err)
		return err
	}
	return writeSSE(w, "state", buildStateSnapshot(records, h.clock()))
}

func writeSSEError(w *bufio.Writer, err error) {
	_ = writeSSE(w, "error", fiber.Map{"error": "state unavailable"})
	_ = w.Flush()
}

func writeSSE(w *bufio.Writer, event string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "event: %s\n", event); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "data: %s\n\n", data); err != nil {
		return err
	}
	return nil
}
