package handlers

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type missionStore interface {
	DeclareMission(ctx context.Context, userID string, declaration string, declarationURL string, now time.Time, missionID string, eventID string) (airlockMission, error)
	SubmitProof(ctx context.Context, userID string, missionID string, proofURL string, now time.Time, eventID string) (airlockMission, error)
	ConfirmProof(ctx context.Context, missionID string, reviewerID string, now time.Time, eventID string) (airlockMission, error)
	RejectProof(ctx context.Context, missionID string, reviewerID string, now time.Time, eventID string) (missionReviewMutation, error)
}

type unavailableMissionStore struct{}

func (unavailableMissionStore) DeclareMission(context.Context, string, string, string, time.Time, string, string) (airlockMission, error) {
	return airlockMission{}, fmt.Errorf("mission store unavailable")
}

func (unavailableMissionStore) SubmitProof(context.Context, string, string, string, time.Time, string) (airlockMission, error) {
	return airlockMission{}, fmt.Errorf("mission store unavailable")
}

func (unavailableMissionStore) ConfirmProof(context.Context, string, string, time.Time, string) (airlockMission, error) {
	return airlockMission{}, fmt.Errorf("mission store unavailable")
}

func (unavailableMissionStore) RejectProof(context.Context, string, string, time.Time, string) (missionReviewMutation, error) {
	return missionReviewMutation{}, fmt.Errorf("mission store unavailable")
}

type dynamoMissionStore struct {
	tableName    string
	region       string
	launchUserID string
	client       dynamoMissionAPI
	s3           s3PutAPI
}

type dynamoMissionAPI interface {
	dynamoScanAPI
	TransactWriteItems(ctx context.Context, params *dynamodb.TransactWriteItemsInput, optFns ...func(*dynamodb.Options)) (*dynamodb.TransactWriteItemsOutput, error)
	UpdateItem(ctx context.Context, params *dynamodb.UpdateItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error)
	GetItem(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error)
}

func newDynamoMissionStore(tableName string, region string, launchUserID string) *dynamoMissionStore {
	return &dynamoMissionStore{tableName: tableName, region: region, launchUserID: launchUserID}
}

func (s *dynamoMissionStore) loadRecords(ctx context.Context, client dynamoMissionAPI) (stateRecords, error) {
	return loadDynamoStateRecords(ctx, client, s.tableName, s.launchUserID)
}

func (s *dynamoMissionStore) DeclareMission(ctx context.Context, userID string, declaration string, declarationURL string, now time.Time, missionID string, eventID string) (airlockMission, error) {
	client, err := s.dynamoClient(ctx)
	if err != nil {
		return airlockMission{}, err
	}
	records, err := s.loadRecords(ctx, client)
	if err != nil {
		return airlockMission{}, err
	}
	mutation, err := buildMissionDeclaration(records, userID, declaration, declarationURL, now, missionID, eventID)
	if err != nil {
		return airlockMission{}, err
	}

	_, err = client.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{
		TransactItems: []types.TransactWriteItem{
			putItem(s.tableName, missionItem(mutation.Mission)),
			putItem(s.tableName, eventItem(mutation.Event)),
		},
	})
	if err != nil {
		if isConditionalTransactionFailure(err) {
			return airlockMission{}, ErrMissionStateConflict
		}
		return airlockMission{}, err
	}
	return mutation.Mission, nil
}

func (s *dynamoMissionStore) SubmitProof(ctx context.Context, userID string, missionID string, proofURL string, now time.Time, eventID string) (airlockMission, error) {
	client, err := s.dynamoClient(ctx)
	if err != nil {
		return airlockMission{}, err
	}
	records, err := s.loadRecords(ctx, client)
	if err != nil {
		return airlockMission{}, err
	}
	mutation, err := buildMissionProof(records, userID, missionID, proofURL, now, eventID)
	if err != nil {
		return airlockMission{}, err
	}

	nowISO := formatInstant(now)
	_, err = client.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{
		TransactItems: []types.TransactWriteItem{
			{
				Update: &types.Update{
					TableName: &s.tableName,
					Key:       missionKey(missionID),
					UpdateExpression: ptrString(
						"SET #status = :pending, #proof_url = :proof_url, #proof_submitted_at = :proof_submitted_at, #gsi2pk = :gsi2pk, #mission_id = :mission_id, #entity = :entity, #version = :version",
					),
					ConditionExpression: ptrString("attribute_exists(#pk) AND #status = :declared AND #deadline_at > :now"),
					ExpressionAttributeNames: map[string]string{
						"#pk":                 "pk",
						"#status":             "status",
						"#deadline_at":        "deadline_at",
						"#proof_url":          "proof_url",
						"#proof_submitted_at": "proof_submitted_at",
						"#gsi2pk":             "gsi2pk",
						"#mission_id":         "mission_id",
						"#entity":             "__edb_e__",
						"#version":            "__edb_v__",
					},
					ExpressionAttributeValues: map[string]types.AttributeValue{
						":pending":            stringAttr("PROOF_PENDING"),
						":declared":           stringAttr("DECLARED"),
						":now":                stringAttr(nowISO),
						":proof_url":          stringAttr(mutation.Mission.ProofURL),
						":proof_submitted_at": stringAttr(mutation.Mission.ProofSubmittedAt),
						":gsi2pk":             stringAttr(electroPK("status", "PROOF_PENDING")),
						":mission_id":         stringAttr(missionID),
						":entity":             stringAttr("mission"),
						":version":            stringAttr("1"),
					},
				},
			},
			putItem(s.tableName, eventItem(mutation.Event)),
		},
	})
	if err != nil {
		if isConditionalTransactionFailure(err) {
			return airlockMission{}, ErrMissionStateConflict
		}
		return airlockMission{}, err
	}
	return mutation.Mission, nil
}

func (s *dynamoMissionStore) ConfirmProof(ctx context.Context, missionID string, reviewerID string, now time.Time, eventID string) (airlockMission, error) {
	client, err := s.dynamoClient(ctx)
	if err != nil {
		return airlockMission{}, err
	}
	records, err := s.loadRecords(ctx, client)
	if err != nil {
		return airlockMission{}, err
	}
	mutation, err := buildMissionConfirmation(records, missionID, reviewerID, now, eventID)
	if err != nil {
		return airlockMission{}, err
	}

	transactItems := []types.TransactWriteItem{
		{
			Update: &types.Update{
				TableName: &s.tableName,
				Key:       missionKey(missionID),
				UpdateExpression: ptrString(
					"SET #status = :confirmed, #confirmed_at = :confirmed_at, #gsi2pk = :gsi2pk, #mission_id = :mission_id, #entity = :entity, #version = :version",
				),
				ConditionExpression: ptrString("attribute_exists(#pk) AND #status = :pending"),
				ExpressionAttributeNames: map[string]string{
					"#pk":           "pk",
					"#status":       "status",
					"#confirmed_at": "confirmed_at",
					"#gsi2pk":       "gsi2pk",
					"#mission_id":   "mission_id",
					"#entity":       "__edb_e__",
					"#version":      "__edb_v__",
				},
				ExpressionAttributeValues: map[string]types.AttributeValue{
					":confirmed":    stringAttr("CONFIRMED"),
					":pending":      stringAttr("PROOF_PENDING"),
					":confirmed_at": stringAttr(mutation.Mission.ConfirmedAt),
					":gsi2pk":       stringAttr(electroPK("status", "CONFIRMED")),
					":mission_id":   stringAttr(missionID),
					":entity":       stringAttr("mission"),
					":version":      stringAttr("1"),
				},
			},
		},
		putItem(s.tableName, eventItem(mutation.Event)),
	}
	for _, redeemedMission := range mutation.RedeemedMissions {
		transactItems = append(transactItems, redeemMissionUpdate(s.tableName, redeemedMission))
	}
	if mutation.RedeemedEvent != nil {
		transactItems = append(transactItems, putItem(s.tableName, eventItem(*mutation.RedeemedEvent)))
	}

	_, err = client.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{
		TransactItems: transactItems,
	})
	if err != nil {
		if isConditionalTransactionFailure(err) {
			return airlockMission{}, ErrMissionStateConflict
		}
		return airlockMission{}, err
	}
	return mutation.Mission, nil
}

func (s *dynamoMissionStore) RejectProof(ctx context.Context, missionID string, reviewerID string, now time.Time, eventID string) (missionReviewMutation, error) {
	client, err := s.dynamoClient(ctx)
	if err != nil {
		return missionReviewMutation{}, err
	}
	records, err := s.loadRecords(ctx, client)
	if err != nil {
		return missionReviewMutation{}, err
	}
	mutation, err := buildMissionRejection(records, missionID, reviewerID, now, eventID)
	if err != nil {
		return missionReviewMutation{}, err
	}

	_, err = client.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{
		TransactItems: []types.TransactWriteItem{
			{
				Update: &types.Update{
					TableName: &s.tableName,
					Key:       missionKey(missionID),
					UpdateExpression: ptrString(
						"SET #status = :airlocked, #airlocked_at = :airlocked_at, #gsi2pk = :mission_gsi2pk, #mission_id = :mission_id, #entity = :mission_entity, #version = :version",
					),
					ConditionExpression: ptrString("attribute_exists(#pk) AND #status = :pending"),
					ExpressionAttributeNames: map[string]string{
						"#pk":           "pk",
						"#status":       "status",
						"#airlocked_at": "airlocked_at",
						"#gsi2pk":       "gsi2pk",
						"#mission_id":   "mission_id",
						"#entity":       "__edb_e__",
						"#version":      "__edb_v__",
					},
					ExpressionAttributeValues: map[string]types.AttributeValue{
						":airlocked":      stringAttr("AIRLOCKED"),
						":pending":        stringAttr("PROOF_PENDING"),
						":airlocked_at":   stringAttr(mutation.Mission.AirlockedAt),
						":mission_gsi2pk": stringAttr(electroPK("status", "AIRLOCKED")),
						":mission_id":     stringAttr(missionID),
						":mission_entity": stringAttr("mission"),
						":version":        stringAttr("1"),
					},
				},
			},
			{
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
						":mission_id":  stringAttr(missionID),
						":seat_gsi2pk": stringAttr(electroPK("status", "VACATED")),
						":seat_id":     stringAttr(mutation.Seat.SeatID),
						":seat_entity": stringAttr("seat"),
						":version":     stringAttr("1"),
					},
				},
			},
			putItem(s.tableName, eventItem(mutation.Event)),
		},
	})
	if err != nil {
		if isConditionalTransactionFailure(err) {
			return missionReviewMutation{}, ErrMissionStateConflict
		}
		return missionReviewMutation{}, err
	}
	return mutation, nil
}

func (s *dynamoMissionStore) dynamoClient(ctx context.Context) (dynamoMissionAPI, error) {
	if s.client != nil {
		return s.client, nil
	}
	cfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(s.region))
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}
	s.client = dynamodb.NewFromConfig(cfg)
	return s.client, nil
}

func putItem(tableName string, item map[string]types.AttributeValue) types.TransactWriteItem {
	return types.TransactWriteItem{
		Put: &types.Put{
			TableName:           &tableName,
			Item:                item,
			ConditionExpression: ptrString("attribute_not_exists(#pk) AND attribute_not_exists(#sk)"),
			ExpressionAttributeNames: map[string]string{
				"#pk": "pk",
				"#sk": "sk",
			},
		},
	}
}

func redeemMissionUpdate(tableName string, mission airlockMission) types.TransactWriteItem {
	return types.TransactWriteItem{
		Update: &types.Update{
			TableName: &tableName,
			Key:       missionKey(mission.MissionID),
			UpdateExpression: ptrString(
				"SET #redeemed = :redeemed, #mission_id = :mission_id, #entity = :entity, #version = :version",
			),
			ConditionExpression: ptrString("attribute_exists(#pk) AND #status = :airlocked"),
			ExpressionAttributeNames: map[string]string{
				"#pk":         "pk",
				"#status":     "status",
				"#redeemed":   "redeemed",
				"#mission_id": "mission_id",
				"#entity":     "__edb_e__",
				"#version":    "__edb_v__",
			},
			ExpressionAttributeValues: map[string]types.AttributeValue{
				":redeemed":   boolAttr(true),
				":airlocked":  stringAttr("AIRLOCKED"),
				":mission_id": stringAttr(mission.MissionID),
				":entity":     stringAttr("mission"),
				":version":    stringAttr("1"),
			},
		},
	}
}

func missionItem(mission airlockMission) map[string]types.AttributeValue {
	item := map[string]types.AttributeValue{
		"entity_type": stringAttr("mission"),
		"mission_id":  stringAttr(mission.MissionID),
		"seat_id":     stringAttr(mission.SeatID),
		"declaration": stringAttr(mission.Declaration),
		"declared_at": stringAttr(mission.DeclaredAt),
		"deadline_at": stringAttr(mission.DeadlineAt),
		"status":      stringAttr(mission.Status),
		"redeemed":    boolAttr(mission.Redeemed),
		"pk":          stringAttr(electroPK("missionid", mission.MissionID)),
		"sk":          stringAttr("$mission_1"),
		"gsi1pk":      stringAttr(electroPK("seatid", mission.SeatID)),
		"gsi1sk":      stringAttr(electroSK("mission", "declaredat", mission.DeclaredAt)),
		"gsi2pk":      stringAttr(electroPK("status", mission.Status)),
		"gsi2sk":      stringAttr(electroSK("mission", "deadlineat", mission.DeadlineAt)),
		"__edb_e__":   stringAttr("mission"),
		"__edb_v__":   stringAttr("1"),
	}
	if mission.DeclarationURL != "" {
		item["declaration_url"] = stringAttr(mission.DeclarationURL)
	}
	return item
}

func eventItem(event airlockEvent) map[string]types.AttributeValue {
	item := map[string]types.AttributeValue{
		"entity_type": stringAttr("event"),
		"event_id":    stringAttr(event.EventID),
		"kind":        stringAttr(event.Kind),
		"user_id":     stringAttr(event.UserID),
		"seat_id":     stringAttr(event.SeatID),
		"occurred_at": stringAttr(event.OccurredAt),
		"message":     stringAttr(event.Message),
		"pk":          stringAttr(electroPK("eventid", event.EventID)),
		"sk":          stringAttr("$event_1"),
		"gsi1pk":      stringAttr(electroPK("kind", event.Kind)),
		"gsi1sk":      stringAttr(electroSK("event", "occurredat", event.OccurredAt)),
		"gsi2pk":      stringAttr(electroPK("userid", event.UserID)),
		"gsi2sk":      stringAttr(electroSK("event", "occurredat", event.OccurredAt)),
		"gsi3pk":      stringAttr(electroPK("seatid", event.SeatID)),
		"gsi3sk":      stringAttr(electroSK("event", "occurredat", event.OccurredAt)),
		"__edb_e__":   stringAttr("event"),
		"__edb_v__":   stringAttr("1"),
	}
	if event.MissionID != "" {
		item["mission_id"] = stringAttr(event.MissionID)
	}
	if event.WakePosted {
		item["wake_posted"] = boolAttr(event.WakePosted)
	}
	return item
}

func missionKey(missionID string) map[string]types.AttributeValue {
	return map[string]types.AttributeValue{
		"pk": stringAttr(electroPK("missionid", missionID)),
		"sk": stringAttr("$mission_1"),
	}
}

func electroPK(field string, value string) string {
	return fmt.Sprintf("$airlock#%s_%s", field, strings.ToLower(value))
}

func electroSK(entity string, field string, value string) string {
	return fmt.Sprintf("$%s_1#%s_%s", entity, field, strings.ToLower(value))
}

func stringAttr(value string) types.AttributeValue {
	return &types.AttributeValueMemberS{Value: value}
}

func boolAttr(value bool) types.AttributeValue {
	return &types.AttributeValueMemberBOOL{Value: value}
}

func ptrString(value string) *string {
	return &value
}

func ptrBool(value bool) *bool {
	return &value
}

func isConditionalTransactionFailure(err error) bool {
	var canceled *types.TransactionCanceledException
	if !errors.As(err, &canceled) {
		return false
	}
	for _, reason := range canceled.CancellationReasons {
		if reason.Code != nil && *reason.Code == "ConditionalCheckFailed" {
			return true
		}
	}
	return false
}
