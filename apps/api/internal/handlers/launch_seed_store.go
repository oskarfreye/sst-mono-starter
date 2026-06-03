package handlers

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

func loadDynamoStateRecords(ctx context.Context, client dynamoStateAPI, tableName string, launchUserID string) (stateRecords, error) {
	records, err := scanStateRecords(ctx, client, tableName)
	if err != nil {
		return stateRecords{}, err
	}
	if launchUserID == "" || hasLaunchSeat(records) {
		return records, nil
	}
	if err := persistLaunchSeed(ctx, client, tableName, launchUserID); err != nil && !isConditionalTransactionFailure(err) {
		return stateRecords{}, err
	}
	return scanStateRecords(ctx, client, tableName)
}

func hasLaunchSeat(records stateRecords) bool {
	for _, seat := range records.Seats {
		if seat.SeatNumber == 1 || seat.SeatID == "SEAT-001" {
			return true
		}
	}
	return false
}

func persistLaunchSeed(ctx context.Context, client dynamoStateAPI, tableName string, launchUserID string) error {
	seed := launchStateRecordsForUserID(launchUserID)
	if len(seed.Users) == 0 || len(seed.Seats) == 0 || len(seed.Stacks) == 0 || len(seed.Missions) == 0 || len(seed.Events) == 0 {
		return fmt.Errorf("launch seed is incomplete")
	}
	launchTime, err := time.Parse(time.RFC3339, launchDeclaredAt)
	if err != nil {
		return fmt.Errorf("parse launch time: %w", err)
	}

	user := seed.Users[0]
	seat := seed.Seats[0]
	stack := seed.Stacks[0]
	mission := seed.Missions[0]
	transactItems := []types.TransactWriteItem{
		putWithCondition(tableName, handleLockItem(user), "attribute_not_exists(#pk) OR #user_id = :user_id", map[string]string{
			"#pk":      "pk",
			"#user_id": "user_id",
		}, map[string]types.AttributeValue{
			":user_id": stringAttr(user.UserID),
		}),
		putWithCondition(tableName, userItem(user, launchTime), "attribute_not_exists(#pk) OR #user_id = :user_id", map[string]string{
			"#pk":      "pk",
			"#user_id": "user_id",
		}, map[string]types.AttributeValue{
			":user_id": stringAttr(user.UserID),
		}),
		putWithCondition(tableName, seatItem(seat), "attribute_not_exists(#pk) OR #occupant_user_id = :user_id", map[string]string{
			"#pk":               "pk",
			"#occupant_user_id": "occupant_user_id",
		}, map[string]types.AttributeValue{
			":user_id": stringAttr(user.UserID),
		}),
		putWithCondition(tableName, stackItemFromStack(stack), "attribute_not_exists(#pk) OR #seat_id = :seat_id", map[string]string{
			"#pk":      "pk",
			"#seat_id": "seat_id",
		}, map[string]types.AttributeValue{
			":seat_id": stringAttr(stack.SeatID),
		}),
		putWithCondition(tableName, missionItem(mission), "attribute_not_exists(#pk) OR #mission_id = :mission_id", map[string]string{
			"#pk":         "pk",
			"#mission_id": "mission_id",
		}, map[string]types.AttributeValue{
			":mission_id": stringAttr(mission.MissionID),
		}),
	}
	for _, event := range seed.Events {
		transactItems = append(transactItems, putWithCondition(tableName, eventItem(event), "attribute_not_exists(#pk) OR #event_id = :event_id", map[string]string{
			"#pk":       "pk",
			"#event_id": "event_id",
		}, map[string]types.AttributeValue{
			":event_id": stringAttr(event.EventID),
		}))
	}

	_, err = client.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{
		TransactItems: transactItems,
	})
	return err
}
