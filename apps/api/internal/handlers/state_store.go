package handlers

import (
	"context"
	"fmt"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type stateStore interface {
	Load(ctx context.Context) (stateRecords, error)
}

type launchStateStore struct{}

func (launchStateStore) Load(context.Context) (stateRecords, error) {
	return launchStateRecords(), nil
}

type dynamoStateStore struct {
	tableName    string
	region       string
	launchUserID string
	client       dynamoStateAPI
}

type dynamoScanAPI interface {
	Scan(ctx context.Context, params *dynamodb.ScanInput, optFns ...func(*dynamodb.Options)) (*dynamodb.ScanOutput, error)
}

type dynamoStateAPI interface {
	dynamoScanAPI
	TransactWriteItems(ctx context.Context, params *dynamodb.TransactWriteItemsInput, optFns ...func(*dynamodb.Options)) (*dynamodb.TransactWriteItemsOutput, error)
}

func newDynamoStateStore(tableName string, region string, launchUserID string) *dynamoStateStore {
	return &dynamoStateStore{tableName: tableName, region: region, launchUserID: launchUserID}
}

func (s *dynamoStateStore) Load(ctx context.Context) (stateRecords, error) {
	client, err := s.dynamoClient(ctx)
	if err != nil {
		return stateRecords{}, err
	}

	records, err := loadDynamoStateRecords(ctx, client, s.tableName, s.launchUserID)
	if err != nil {
		return stateRecords{}, err
	}
	if s.launchUserID == "" {
		if len(records.Users) == 0 && len(records.Seats) == 0 && len(records.Stacks) == 0 && len(records.Missions) == 0 && len(records.Events) == 0 {
			records = launchStateRecords()
			records.Source = "dynamodb_empty_launch_seed"
		} else {
			records = ensureLaunchSeat(records)
		}
	}
	return records, nil
}

func scanStateRecords(ctx context.Context, client dynamoScanAPI, tableName string) (stateRecords, error) {
	var records stateRecords
	records.Source = "dynamodb"
	var startKey map[string]types.AttributeValue
	for {
		out, err := client.Scan(ctx, &dynamodb.ScanInput{
			TableName:         &tableName,
			ConsistentRead:    ptrBool(true),
			ExclusiveStartKey: startKey,
		})
		if err != nil {
			return stateRecords{}, err
		}

		for _, item := range out.Items {
			if err := appendStateRecord(&records, item); err != nil {
				return stateRecords{}, err
			}
		}
		if len(out.LastEvaluatedKey) == 0 {
			break
		}
		startKey = out.LastEvaluatedKey
	}

	return records, nil
}

func (s *dynamoStateStore) dynamoClient(ctx context.Context) (dynamoStateAPI, error) {
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

func appendStateRecord(records *stateRecords, item map[string]types.AttributeValue) error {
	var tag struct {
		EntityType string `dynamodbav:"entity_type"`
	}
	if err := attributevalue.UnmarshalMap(item, &tag); err != nil {
		return fmt.Errorf("decode entity tag: %w", err)
	}

	switch tag.EntityType {
	case "user":
		var record dynamoUserRecord
		if err := attributevalue.UnmarshalMap(item, &record); err != nil {
			return fmt.Errorf("decode user: %w", err)
		}
		records.Users = append(records.Users, airlockUser(record))
	case "seat":
		var record dynamoSeatRecord
		if err := attributevalue.UnmarshalMap(item, &record); err != nil {
			return fmt.Errorf("decode seat: %w", err)
		}
		records.Seats = append(records.Seats, airlockSeat(record))
	case "mission":
		var record dynamoMissionRecord
		if err := attributevalue.UnmarshalMap(item, &record); err != nil {
			return fmt.Errorf("decode mission: %w", err)
		}
		records.Missions = append(records.Missions, airlockMission(record))
	case "stack":
		var record dynamoStackRecord
		if err := attributevalue.UnmarshalMap(item, &record); err != nil {
			return fmt.Errorf("decode stack: %w", err)
		}
		records.Stacks = append(records.Stacks, airlockStack(record))
	case "event":
		var record dynamoEventRecord
		if err := attributevalue.UnmarshalMap(item, &record); err != nil {
			return fmt.Errorf("decode event: %w", err)
		}
		records.Events = append(records.Events, airlockEvent(record))
	default:
		// Existing starter entities or unrelated future records can share the
		// table. Ignore anything outside the Airlock state model.
	}
	return nil
}

func ensureLaunchSeat(records stateRecords) stateRecords {
	for _, seat := range records.Seats {
		if seat.SeatNumber == 1 {
			return records
		}
	}
	seed := launchStateRecords()
	records.Users = append(seed.Users, records.Users...)
	records.Seats = append(seed.Seats, records.Seats...)
	records.Stacks = append(seed.Stacks, records.Stacks...)
	records.Missions = append(seed.Missions, records.Missions...)
	records.Events = append(seed.Events, records.Events...)
	if records.Source == "dynamodb" {
		records.Source = "dynamodb_with_launch_seed"
	}
	return records
}

type dynamoUserRecord struct {
	UserID       string `dynamodbav:"user_id"`
	Handle       string `dynamodbav:"handle"`
	DisplayName  string `dynamodbav:"display_name"`
	Email        string `dynamodbav:"email"`
	AvatarSeed   string `dynamodbav:"avatar_seed"`
	Bio          string `dynamodbav:"bio"`
	AvatarURL    string `dynamodbav:"avatar_url"`
	AvatarStatus string `dynamodbav:"avatar_status"`
}

type dynamoSeatRecord struct {
	SeatID               string `dynamodbav:"seat_id"`
	SeatNumber           int    `dynamodbav:"seat_number"`
	Cohort               string `dynamodbav:"cohort"`
	OccupantUserID       string `dynamodbav:"occupant_user_id"`
	Status               string `dynamodbav:"status"`
	TierPaid             int    `dynamodbav:"tier_paid"`
	PricePaidCents       int    `dynamodbav:"price_paid"`
	BoardedAt            string `dynamodbav:"boarded_at"`
	VacatedAt            string `dynamodbav:"vacated_at"`
	VacatedOnMissionID   string `dynamodbav:"vacated_on_mission_id"`
	ReboardedAsSeatID    string `dynamodbav:"reboarded_as_seat_id"`
	ReboardedAsSeatLabel string `dynamodbav:"reboarded_as_seat_label"`
}

type dynamoMissionRecord struct {
	MissionID        string `dynamodbav:"mission_id"`
	SeatID           string `dynamodbav:"seat_id"`
	Declaration      string `dynamodbav:"declaration"`
	DeclarationURL   string `dynamodbav:"declaration_url"`
	DeclaredAt       string `dynamodbav:"declared_at"`
	DeadlineAt       string `dynamodbav:"deadline_at"`
	Status           string `dynamodbav:"status"`
	ProofURL         string `dynamodbav:"proof_url"`
	ProofSubmittedAt string `dynamodbav:"proof_submitted_at"`
	ConfirmedAt      string `dynamodbav:"confirmed_at"`
	AirlockedAt      string `dynamodbav:"airlocked_at"`
	Redeemed         bool   `dynamodbav:"redeemed"`
}

type dynamoStackRecord struct {
	SeatID                  string   `dynamodbav:"seat_id"`
	DonationEnabled         bool     `dynamodbav:"donation_enabled"`
	DonationAmount          int      `dynamodbav:"donation_amount"`
	DonationTarget          string   `dynamodbav:"donation_target"`
	CrewAlertEnabled        bool     `dynamodbav:"crew_alert_enabled"`
	CrewAlertContacts       []string `dynamodbav:"crew_alert_contacts"`
	TheWakeEnabled          bool     `dynamodbav:"the_wake_enabled"`
	TheWakeCrosspost        []string `dynamodbav:"the_wake_crosspost"`
	ReentryChallengeEnabled bool     `dynamodbav:"reentry_challenge_enabled"`
}

type dynamoEventRecord struct {
	EventID    string `dynamodbav:"event_id"`
	Kind       string `dynamodbav:"kind"`
	UserID     string `dynamodbav:"user_id"`
	SeatID     string `dynamodbav:"seat_id"`
	MissionID  string `dynamodbav:"mission_id"`
	OccurredAt string `dynamodbav:"occurred_at"`
	Message    string `dynamodbav:"message"`
	WakePosted bool   `dynamodbav:"wake_posted"`
}
