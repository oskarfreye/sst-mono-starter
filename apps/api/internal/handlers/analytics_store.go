package handlers

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

func profileViewKey(handle string) map[string]types.AttributeValue {
	return map[string]types.AttributeValue{
		"pk": stringAttr(electroPK("handle", handle)),
		"sk": stringAttr("$profileview_1"),
	}
}

// IncrementProfileView bumps the per-handle view counter, lazily creating the
// counter item on first hit (ADD initializes a missing numeric attribute).
func (s *dynamoMissionStore) IncrementProfileView(ctx context.Context, handle string, now time.Time) error {
	client, err := s.dynamoClient(ctx)
	if err != nil {
		return err
	}
	_, err = client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: &s.tableName,
		Key:       profileViewKey(handle),
		UpdateExpression: ptrString(
			"ADD #views :one SET #entity_type = :etype, #handle = :handle, #updated_at = :updated, #edbe = :edbe, #edbv = :edbv",
		),
		ExpressionAttributeNames: map[string]string{
			"#views":       "views",
			"#entity_type": "entity_type",
			"#handle":      "handle",
			"#updated_at":  "updated_at",
			"#edbe":        "__edb_e__",
			"#edbv":        "__edb_v__",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":one":     &types.AttributeValueMemberN{Value: "1"},
			":etype":   stringAttr("profile_view"),
			":handle":  stringAttr(handle),
			":updated": stringAttr(formatInstant(now)),
			":edbe":    stringAttr("profileView"),
			":edbv":    stringAttr("1"),
		},
	})
	if err != nil {
		return fmt.Errorf("increment profile view: %w", err)
	}
	return nil
}

// GetProfileViews reads the per-handle counter, treating a missing item as 0.
func (s *dynamoMissionStore) GetProfileViews(ctx context.Context, handle string) (int, error) {
	client, err := s.dynamoClient(ctx)
	if err != nil {
		return 0, err
	}
	out, err := client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: &s.tableName,
		Key:       profileViewKey(handle),
	})
	if err != nil {
		return 0, fmt.Errorf("get profile views: %w", err)
	}
	if out.Item == nil {
		return 0, nil
	}
	views, ok := out.Item["views"].(*types.AttributeValueMemberN)
	if !ok {
		return 0, nil
	}
	value, err := strconv.Atoi(views.Value)
	if err != nil {
		return 0, fmt.Errorf("parse profile views: %w", err)
	}
	return value, nil
}
