package handlers

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

func TestLoadDynamoStateRecordsPersistsLaunchSeedWhenConfigured(t *testing.T) {
	api := &fakeDynamoStateAPI{}

	records, err := loadDynamoStateRecords(context.Background(), api, "airlock-test", "USER-REAL-OSKAR")
	if err != nil {
		t.Fatalf("loadDynamoStateRecords returned error: %v", err)
	}

	if api.transacts != 1 {
		t.Fatalf("transacts = %d, want one launch seed write", api.transacts)
	}
	if len(records.Seats) != 1 || records.Seats[0].SeatID != "SEAT-001" || records.Seats[0].OccupantUserID != "USER-REAL-OSKAR" {
		t.Fatalf("seats = %#v, want persisted launch seat owned by configured user", records.Seats)
	}
	if len(records.Missions) != 1 || records.Missions[0].DeclarationURL != "https://theairlock.space" {
		t.Fatalf("missions = %#v, want launch mission with declaration URL", records.Missions)
	}
}

func TestLoadDynamoStateRecordsDoesNotPersistLaunchSeedWhenSeatExists(t *testing.T) {
	api := &fakeDynamoStateAPI{
		items: []map[string]types.AttributeValue{
			seatItem(airlockSeat{
				SeatID:         "SEAT-001",
				SeatNumber:     1,
				Cohort:         "THE_100",
				OccupantUserID: "USER-REAL-OSKAR",
				Status:         "OCCUPIED",
				BoardedAt:      launchDeclaredAt,
			}),
		},
	}

	records, err := loadDynamoStateRecords(context.Background(), api, "airlock-test", "USER-REAL-OSKAR")
	if err != nil {
		t.Fatalf("loadDynamoStateRecords returned error: %v", err)
	}

	if api.transacts != 0 {
		t.Fatalf("transacts = %d, want no launch seed write", api.transacts)
	}
	if len(records.Seats) != 1 || records.Seats[0].SeatID != "SEAT-001" {
		t.Fatalf("seats = %#v, want existing launch seat", records.Seats)
	}
}

type fakeDynamoStateAPI struct {
	items     []map[string]types.AttributeValue
	transacts int
}

func (f *fakeDynamoStateAPI) Scan(context.Context, *dynamodb.ScanInput, ...func(*dynamodb.Options)) (*dynamodb.ScanOutput, error) {
	return &dynamodb.ScanOutput{Items: f.items}, nil
}

func (f *fakeDynamoStateAPI) TransactWriteItems(_ context.Context, params *dynamodb.TransactWriteItemsInput, _ ...func(*dynamodb.Options)) (*dynamodb.TransactWriteItemsOutput, error) {
	f.transacts++
	for _, item := range params.TransactItems {
		if item.Put != nil {
			f.items = append(f.items, item.Put.Item)
		}
	}
	return &dynamodb.TransactWriteItemsOutput{}, nil
}
