package handlers

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type stackStore interface {
	LoadStack(ctx context.Context, userID string, now time.Time) (airlockStack, error)
	UpdateStack(ctx context.Context, userID string, input stackUpdateInput, now time.Time) (airlockStack, error)
}

type unavailableStackStore struct{}

func (unavailableStackStore) LoadStack(context.Context, string, time.Time) (airlockStack, error) {
	return airlockStack{}, fmt.Errorf("stack store unavailable")
}

func (unavailableStackStore) UpdateStack(context.Context, string, stackUpdateInput, time.Time) (airlockStack, error) {
	return airlockStack{}, fmt.Errorf("stack store unavailable")
}

func (s *dynamoMissionStore) LoadStack(ctx context.Context, userID string, now time.Time) (airlockStack, error) {
	client, err := s.dynamoClient(ctx)
	if err != nil {
		return airlockStack{}, err
	}
	records, err := s.loadRecords(ctx, client)
	if err != nil {
		return airlockStack{}, err
	}
	return stackForUser(records, userID, now)
}

func (s *dynamoMissionStore) UpdateStack(ctx context.Context, userID string, input stackUpdateInput, now time.Time) (airlockStack, error) {
	client, err := s.dynamoClient(ctx)
	if err != nil {
		return airlockStack{}, err
	}
	records, err := s.loadRecords(ctx, client)
	if err != nil {
		return airlockStack{}, err
	}
	stack, err := buildStackUpdate(records, userID, input, now)
	if err != nil {
		return airlockStack{}, err
	}

	_, err = client.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{
		TransactItems: []types.TransactWriteItem{
			{
				ConditionCheck: &types.ConditionCheck{
					TableName:           &s.tableName,
					Key:                 seatKey(stack.SeatID),
					ConditionExpression: ptrString("attribute_exists(#pk) AND #status = :occupied AND #occupant_user_id = :user_id"),
					ExpressionAttributeNames: map[string]string{
						"#pk":               "pk",
						"#status":           "status",
						"#occupant_user_id": "occupant_user_id",
					},
					ExpressionAttributeValues: map[string]types.AttributeValue{
						":occupied": stringAttr("OCCUPIED"),
						":user_id":  stringAttr(userID),
					},
				},
			},
			putWithCondition(s.tableName, stackItemFromStack(stack), "attribute_not_exists(#pk) OR #seat_id = :seat_id", map[string]string{
				"#pk":      "pk",
				"#seat_id": "seat_id",
			}, map[string]types.AttributeValue{
				":seat_id": stringAttr(stack.SeatID),
			}),
		},
	})
	if err != nil {
		if isConditionalTransactionFailure(err) {
			return airlockStack{}, ErrMissionStateConflict
		}
		return airlockStack{}, err
	}
	return stack, nil
}

func stackItemFromStack(stack airlockStack) map[string]types.AttributeValue {
	return map[string]types.AttributeValue{
		"entity_type":               stringAttr("stack"),
		"seat_id":                   stringAttr(stack.SeatID),
		"donation_enabled":          boolAttr(false),
		"crew_alert_enabled":        boolAttr(stack.CrewAlertEnabled),
		"crew_alert_contacts":       stringListAttr(stack.CrewAlertContacts),
		"the_wake_enabled":          boolAttr(stack.TheWakeEnabled),
		"the_wake_crosspost":        stringListAttr(stack.TheWakeCrosspost),
		"reentry_challenge_enabled": boolAttr(stack.ReentryChallengeEnabled),
		"pk":                        stringAttr(electroPK("seatid", stack.SeatID)),
		"sk":                        stringAttr("$stack_1"),
		"__edb_e__":                 stringAttr("stack"),
		"__edb_v__":                 stringAttr("1"),
	}
}

func stringListAttr(values []string) types.AttributeValue {
	out := make([]types.AttributeValue, 0, len(values))
	for _, value := range values {
		out = append(out, stringAttr(value))
	}
	return &types.AttributeValueMemberL{Value: out}
}
