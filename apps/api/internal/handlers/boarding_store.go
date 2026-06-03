package handlers

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type boardingStore interface {
	Board(ctx context.Context, userID string, email string, handle string, displayName string, planAmountCents int, now time.Time) (boardingResult, error)
}

type boardingResult struct {
	SeatID     string
	SeatNumber int
	SeatLabel  string
	Cohort     string
	Handle     string
	Created    bool
}

type unavailableBoardingStore struct{}

func (unavailableBoardingStore) Board(context.Context, string, string, string, string, int, time.Time) (boardingResult, error) {
	return boardingResult{}, fmt.Errorf("boarding store unavailable")
}

func (s *dynamoMissionStore) Board(ctx context.Context, userID string, email string, handle string, displayName string, planAmountCents int, now time.Time) (boardingResult, error) {
	client, err := s.dynamoClient(ctx)
	if err != nil {
		return boardingResult{}, err
	}
	for attempt := 0; attempt < 5; attempt++ {
		records, err := s.loadRecords(ctx, client)
		if err != nil {
			return boardingResult{}, err
		}

		// Idempotent: a user that already holds an occupied (non-vacated) seat
		// gets that seat back without re-boarding.
		if seat, ok := occupiedSeatForUser(records, userID); ok {
			return boardingResult{
				SeatID:     seat.SeatID,
				SeatNumber: seat.SeatNumber,
				SeatLabel:  seatLabel(seat.SeatNumber),
				Cohort:     cohortForSeat(seat.SeatNumber, seat.Cohort),
				Handle:     handleForSeat(records, userID, seat),
				Created:    false,
			}, nil
		}

		fulfillment, err := buildBoarding(records, userID, email, handle, displayName, planAmountCents, now)
		if err != nil {
			return boardingResult{}, err
		}

		err = s.putBoardingFulfillment(ctx, client, fulfillment, now)
		if err == nil {
			return boardingResult{
				SeatID:     fulfillment.Seat.SeatID,
				SeatNumber: fulfillment.Seat.SeatNumber,
				SeatLabel:  seatLabel(fulfillment.Seat.SeatNumber),
				Cohort:     fulfillment.Seat.Cohort,
				Handle:     fulfillment.User.Handle,
				Created:    true,
			}, nil
		}
		if isConditionalTransactionFailure(err) {
			continue
		}
		return boardingResult{}, err
	}
	return boardingResult{}, ErrMissionStateConflict
}

func occupiedSeatForUser(records stateRecords, userID string) (airlockSeat, bool) {
	for _, seat := range records.Seats {
		if seat.OccupantUserID != userID {
			continue
		}
		if normalizeStatus(seat.Status) == "OCCUPIED" && seat.VacatedAt == "" {
			return seat, true
		}
	}
	return airlockSeat{}, false
}

func handleForSeat(records stateRecords, userID string, seat airlockSeat) string {
	for _, user := range records.Users {
		if user.UserID == userID && user.Handle != "" {
			return user.Handle
		}
	}
	return seat.SeatID
}

func (s *dynamoMissionStore) putBoardingFulfillment(ctx context.Context, client dynamoMissionAPI, fulfillment boardingFulfillment, now time.Time) error {
	user := fulfillment.User
	seat := fulfillment.Seat
	event := fulfillment.Event
	transactItems := []types.TransactWriteItem{
		putWithCondition(s.tableName, handleLockItem(user), "attribute_not_exists(#pk) OR #user_id = :user_id", map[string]string{
			"#pk":      "pk",
			"#user_id": "user_id",
		}, map[string]types.AttributeValue{
			":user_id": stringAttr(user.UserID),
		}),
		putWithCondition(s.tableName, userItem(user, now), "attribute_not_exists(#pk) OR #user_id = :user_id", map[string]string{
			"#pk":      "pk",
			"#user_id": "user_id",
		}, map[string]types.AttributeValue{
			":user_id": stringAttr(user.UserID),
		}),
		seatCounterUpdate(s.tableName, seat.SeatNumber),
	}
	for _, linkedSeat := range fulfillment.LinkedVacatedSeats {
		transactItems = append(transactItems, reboardedSeatUpdate(s.tableName, linkedSeat))
	}
	transactItems = append(transactItems,
		putItem(s.tableName, seatItem(seat)),
		putItem(s.tableName, stackItem(seat.SeatID)),
		putItem(s.tableName, eventItem(event)),
	)

	_, err := client.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{
		TransactItems: transactItems,
	})
	return err
}

func userItem(user airlockUser, now time.Time) map[string]types.AttributeValue {
	if user.AvatarSeed == "" {
		user.AvatarSeed = user.Handle
	}
	// Deliberately omit avatar_url/avatar_status/avatar_updated_at: boarding (re)creates
	// the user with no avatar so re-boarding resets to the visor-down default until re-upload.
	return map[string]types.AttributeValue{
		"entity_type":  stringAttr("user"),
		"user_id":      stringAttr(user.UserID),
		"handle":       stringAttr(user.Handle),
		"display_name": stringAttr(user.DisplayName),
		"email":        stringAttr(user.Email),
		"joined_at":    stringAttr(formatInstant(now)),
		"avatar_seed":  stringAttr(user.AvatarSeed),
		"pk":           stringAttr(electroPK("userid", user.UserID)),
		"sk":           stringAttr("$user_1"),
		"gsi1pk":       stringAttr(electroPK("handle", user.Handle)),
		"gsi1sk":       stringAttr("$user_1"),
		"__edb_e__":    stringAttr("user"),
		"__edb_v__":    stringAttr("1"),
	}
}

func handleLockItem(user airlockUser) map[string]types.AttributeValue {
	return map[string]types.AttributeValue{
		"entity_type": stringAttr("handle"),
		"handle":      stringAttr(user.Handle),
		"user_id":     stringAttr(user.UserID),
		"pk":          stringAttr(electroPK("handle", user.Handle)),
		"sk":          stringAttr("$handle_1"),
	}
}

func seatItem(seat airlockSeat) map[string]types.AttributeValue {
	item := map[string]types.AttributeValue{
		"entity_type":      stringAttr("seat"),
		"seat_id":          stringAttr(seat.SeatID),
		"seat_number":      numberAttr(seat.SeatNumber),
		"cohort":           stringAttr(seat.Cohort),
		"occupant_user_id": stringAttr(seat.OccupantUserID),
		"status":           stringAttr(seat.Status),
		"tier_paid":        numberAttr(seat.TierPaid),
		"price_paid":       numberAttr(seat.PricePaidCents),
		"boarded_at":       stringAttr(seat.BoardedAt),
		"pk":               stringAttr(electroPK("seatid", seat.SeatID)),
		"sk":               stringAttr("$seat_1"),
		"gsi1pk":           stringAttr(electroPK("occupantuserid", seat.OccupantUserID)),
		"gsi1sk":           stringAttr(electroSK("seat", "boardedat", seat.BoardedAt)),
		"gsi2pk":           stringAttr(electroPK("status", seat.Status)),
		"gsi2sk":           stringAttr(electroSK("seat", "boardedat", seat.BoardedAt)),
		"__edb_e__":        stringAttr("seat"),
		"__edb_v__":        stringAttr("1"),
	}
	if seat.ReboardedAsSeatID != "" {
		item["reboarded_as_seat_id"] = stringAttr(seat.ReboardedAsSeatID)
	}
	if seat.ReboardedAsSeatLabel != "" {
		item["reboarded_as_seat_label"] = stringAttr(seat.ReboardedAsSeatLabel)
	}
	return item
}

func stackItem(seatID string) map[string]types.AttributeValue {
	return stackItemFromStack(airlockStack{SeatID: seatID})
}

func seatCounterUpdate(tableName string, seatNumber int) types.TransactWriteItem {
	previousSeatNumber := seatNumber - 1
	return types.TransactWriteItem{
		Update: &types.Update{
			TableName: &tableName,
			Key:       seatCounterKey(),
			UpdateExpression: ptrString(
				"SET #entity_type = :entity_type, #counter_id = :counter_id, #last_seat_number = :seat_number, #entity = :entity, #version = :version",
			),
			ConditionExpression: ptrString("attribute_not_exists(#pk) OR #last_seat_number <= :previous_seat_number"),
			ExpressionAttributeNames: map[string]string{
				"#pk":               "pk",
				"#entity_type":      "entity_type",
				"#counter_id":       "counter_id",
				"#last_seat_number": "last_seat_number",
				"#entity":           "__edb_e__",
				"#version":          "__edb_v__",
			},
			ExpressionAttributeValues: map[string]types.AttributeValue{
				":entity_type":          stringAttr("counter"),
				":counter_id":           stringAttr("seats"),
				":seat_number":          numberAttr(seatNumber),
				":previous_seat_number": numberAttr(previousSeatNumber),
				":entity":               stringAttr("counter"),
				":version":              stringAttr("1"),
			},
		},
	}
}

func seatCounterKey() map[string]types.AttributeValue {
	return map[string]types.AttributeValue{
		"pk": stringAttr(electroPK("counterid", "seats")),
		"sk": stringAttr("$counter_1"),
	}
}

func reboardedSeatUpdate(tableName string, seat airlockSeat) types.TransactWriteItem {
	return types.TransactWriteItem{
		Update: &types.Update{
			TableName: &tableName,
			Key:       seatKey(seat.SeatID),
			UpdateExpression: ptrString(
				"SET #reboarded_as_seat_id = :reboarded_as_seat_id, #reboarded_as_seat_label = :reboarded_as_seat_label, #seat_id = :seat_id, #entity = :entity, #version = :version",
			),
			ConditionExpression: ptrString("attribute_exists(#pk) AND #status = :vacated AND attribute_not_exists(#reboarded_as_seat_id)"),
			ExpressionAttributeNames: map[string]string{
				"#pk":                      "pk",
				"#status":                  "status",
				"#reboarded_as_seat_id":    "reboarded_as_seat_id",
				"#reboarded_as_seat_label": "reboarded_as_seat_label",
				"#seat_id":                 "seat_id",
				"#entity":                  "__edb_e__",
				"#version":                 "__edb_v__",
			},
			ExpressionAttributeValues: map[string]types.AttributeValue{
				":vacated":                 stringAttr("VACATED"),
				":reboarded_as_seat_id":    stringAttr(seat.ReboardedAsSeatID),
				":reboarded_as_seat_label": stringAttr(seat.ReboardedAsSeatLabel),
				":seat_id":                 stringAttr(seat.SeatID),
				":entity":                  stringAttr("seat"),
				":version":                 stringAttr("1"),
			},
		},
	}
}

func putWithCondition(tableName string, item map[string]types.AttributeValue, condition string, names map[string]string, values map[string]types.AttributeValue) types.TransactWriteItem {
	return types.TransactWriteItem{
		Put: &types.Put{
			TableName:                 &tableName,
			Item:                      item,
			ConditionExpression:       ptrString(condition),
			ExpressionAttributeNames:  names,
			ExpressionAttributeValues: values,
		},
	}
}

func numberAttr(value int) types.AttributeValue {
	return &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", value)}
}
