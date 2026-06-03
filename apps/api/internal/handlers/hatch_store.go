package handlers

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type hatchStore interface {
	RunHatch(ctx context.Context, now time.Time, newID func(prefix string) string) (HatchRunResult, error)
}

type HatchRunResult struct {
	Scanned   int             `json:"scanned"`
	Airlocked int             `json:"airlocked"`
	Skipped   int             `json:"skipped"`
	Events    []HatchRunEvent `json:"events"`
}

type HatchRunEvent struct {
	EventID           string   `json:"event_id"`
	Handle            string   `json:"handle"`
	SeatID            string   `json:"seat_id"`
	MissionID         string   `json:"mission_id"`
	Mission           string   `json:"mission"`
	OccurredAt        string   `json:"occurred_at"`
	CrewAlertContacts []string `json:"crew_alert_contacts,omitempty"`
	CrewAlertSent     bool     `json:"crew_alert_sent,omitempty"`
	CrewAlertError    string   `json:"crew_alert_error,omitempty"`
	WakePosted        bool     `json:"wake_posted"`
}

type unavailableHatchStore struct{}

func (unavailableHatchStore) RunHatch(context.Context, time.Time, func(string) string) (HatchRunResult, error) {
	return HatchRunResult{}, ErrMissionStateConflict
}

func (s *dynamoMissionStore) RunHatch(ctx context.Context, now time.Time, newID func(string) string) (HatchRunResult, error) {
	client, err := s.dynamoClient(ctx)
	if err != nil {
		return HatchRunResult{}, err
	}
	records, err := s.loadRecords(ctx, client)
	if err != nil {
		return HatchRunResult{}, err
	}

	mutations := buildHatchMutations(records, now, newID)
	result := HatchRunResult{
		Scanned: hatchScanCount(records),
		Events:  make([]HatchRunEvent, 0, len(mutations)),
	}
	for _, mutation := range mutations {
		if err := s.applyHatchMutation(ctx, client, mutation, now); err != nil {
			if isConditionalTransactionFailure(err) {
				result.Skipped++
				continue
			}
			return result, err
		}
		result.Airlocked++
		result.Events = append(result.Events, HatchRunEvent{
			EventID:           mutation.Event.EventID,
			Handle:            mutation.Handle,
			SeatID:            mutation.Seat.SeatID,
			MissionID:         mutation.Mission.MissionID,
			Mission:           mutation.Mission.Declaration,
			OccurredAt:        mutation.Event.OccurredAt,
			CrewAlertContacts: mutation.CrewAlertContacts,
			WakePosted:        mutation.WakePosted,
		})
	}
	return result, nil
}

func (s *dynamoMissionStore) applyHatchMutation(ctx context.Context, client dynamoMissionAPI, mutation hatchMutation, now time.Time) error {
	nowISO := formatInstant(now)
	transactItems := []types.TransactWriteItem{}
	if mutation.CreateMission {
		transactItems = append(transactItems, putItem(s.tableName, missionItem(mutation.Mission)))
	} else {
		transactItems = append(transactItems,
			types.TransactWriteItem{
				Update: &types.Update{
					TableName: &s.tableName,
					Key:       missionKey(mutation.Mission.MissionID),
					UpdateExpression: ptrString(
						"SET #status = :airlocked, #airlocked_at = :airlocked_at, #gsi2pk = :mission_gsi2pk, #mission_id = :mission_id, #entity = :mission_entity, #version = :version",
					),
					ConditionExpression: ptrString("attribute_exists(#pk) AND (#status = :declared OR #status = :pending) AND #deadline_at < :now"),
					ExpressionAttributeNames: map[string]string{
						"#pk":           "pk",
						"#status":       "status",
						"#deadline_at":  "deadline_at",
						"#airlocked_at": "airlocked_at",
						"#gsi2pk":       "gsi2pk",
						"#mission_id":   "mission_id",
						"#entity":       "__edb_e__",
						"#version":      "__edb_v__",
					},
					ExpressionAttributeValues: map[string]types.AttributeValue{
						":airlocked":      stringAttr("AIRLOCKED"),
						":declared":       stringAttr("DECLARED"),
						":pending":        stringAttr("PROOF_PENDING"),
						":now":            stringAttr(nowISO),
						":airlocked_at":   stringAttr(mutation.Mission.AirlockedAt),
						":mission_gsi2pk": stringAttr(electroPK("status", "AIRLOCKED")),
						":mission_id":     stringAttr(mutation.Mission.MissionID),
						":mission_entity": stringAttr("mission"),
						":version":        stringAttr("1"),
					},
				},
			},
		)
	}
	transactItems = append(transactItems,
		types.TransactWriteItem{
			Update: &types.Update{
				TableName: &s.tableName,
				Key:       seatKey(mutation.Seat.SeatID),
				UpdateExpression: ptrString(
					"SET #status = :vacated, #vacated_at = :vacated_at, #vacated_on_mission_id = :mission_id, #gsi2pk = :seat_gsi2pk, #seat_id = :seat_id, #entity = :seat_entity, #version = :version",
				),
				ConditionExpression: ptrString("attribute_exists(#pk) AND #status = :occupied"),
				ExpressionAttributeNames: map[string]string{
					"#pk":                    "pk",
					"#status":                "status",
					"#vacated_at":            "vacated_at",
					"#vacated_on_mission_id": "vacated_on_mission_id",
					"#gsi2pk":                "gsi2pk",
					"#seat_id":               "seat_id",
					"#entity":                "__edb_e__",
					"#version":               "__edb_v__",
				},
				ExpressionAttributeValues: map[string]types.AttributeValue{
					":vacated":     stringAttr("VACATED"),
					":occupied":    stringAttr("OCCUPIED"),
					":vacated_at":  stringAttr(mutation.Seat.VacatedAt),
					":mission_id":  stringAttr(mutation.Mission.MissionID),
					":seat_gsi2pk": stringAttr(electroPK("status", "VACATED")),
					":seat_id":     stringAttr(mutation.Seat.SeatID),
					":seat_entity": stringAttr("seat"),
					":version":     stringAttr("1"),
				},
			},
		},
		putItem(s.tableName, eventItem(mutation.Event)),
	)

	_, err := client.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{
		TransactItems: transactItems,
	})
	return err
}

func seatKey(seatID string) map[string]types.AttributeValue {
	return map[string]types.AttributeValue{
		"pk": stringAttr(electroPK("seatid", seatID)),
		"sk": stringAttr("$seat_1"),
	}
}
