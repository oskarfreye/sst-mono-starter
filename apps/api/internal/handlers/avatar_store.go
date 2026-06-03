package handlers

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type s3PutAPI interface {
	PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
}

// s3Client lazily builds an S3 client against the store's AWS_REGION, mirroring
// the dynamoClient pattern (the buckets live in the Lambda's home region).
func (s *dynamoMissionStore) s3Client(ctx context.Context) (s3PutAPI, error) {
	if s.s3 != nil {
		return s.s3, nil
	}
	cfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(s.region))
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}
	s.s3 = s3.NewFromConfig(cfg)
	return s.s3, nil
}

// PutAvatarSource stores the user-uploaded source image in the private bucket.
func (s *dynamoMissionStore) PutAvatarSource(ctx context.Context, bucket string, userID string, data []byte, contentType string) error {
	client, err := s.s3Client(ctx)
	if err != nil {
		return err
	}
	_, err = client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String("avatar-src/" + userID),
		Body:        bytes.NewReader(data),
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return fmt.Errorf("put avatar source: %w", err)
	}
	return nil
}

// PutAvatarPublic stores the generated PNG in the public bucket under the
// CDN-served key avatars/<userID>.png.
func (s *dynamoMissionStore) PutAvatarPublic(ctx context.Context, bucket string, userID string, png []byte) error {
	client, err := s.s3Client(ctx)
	if err != nil {
		return err
	}
	_, err = client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:       aws.String(bucket),
		Key:          aws.String("avatars/" + userID + ".png"),
		Body:         bytes.NewReader(png),
		ContentType:  aws.String("image/png"),
		CacheControl: aws.String("public, max-age=300"),
	})
	if err != nil {
		return fmt.Errorf("put avatar public: %w", err)
	}
	return nil
}

func userItemKey(userID string) map[string]types.AttributeValue {
	return map[string]types.AttributeValue{
		"pk": stringAttr(electroPK("userid", userID)),
		"sk": stringAttr("$user_1"),
	}
}

// SetUserAvatar marks the user's avatar as ready, recording the CDN url.
func (s *dynamoMissionStore) SetUserAvatar(ctx context.Context, userID string, url string, status string, now time.Time) error {
	client, err := s.dynamoClient(ctx)
	if err != nil {
		return err
	}
	_, err = client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName:           &s.tableName,
		Key:                 userItemKey(userID),
		UpdateExpression:    ptrString("SET #avatar_url = :avatar_url, #avatar_status = :avatar_status, #avatar_updated_at = :avatar_updated_at"),
		ConditionExpression: ptrString("attribute_exists(#pk)"),
		ExpressionAttributeNames: map[string]string{
			"#pk":                "pk",
			"#avatar_url":        "avatar_url",
			"#avatar_status":     "avatar_status",
			"#avatar_updated_at": "avatar_updated_at",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":avatar_url":        stringAttr(url),
			":avatar_status":     stringAttr(status),
			":avatar_updated_at": stringAttr(formatInstant(now)),
		},
	})
	if err != nil {
		return fmt.Errorf("set user avatar: %w", err)
	}
	return nil
}

// ClearUserAvatar removes the avatar url and resets the status to NONE.
func (s *dynamoMissionStore) ClearUserAvatar(ctx context.Context, userID string, now time.Time) error {
	client, err := s.dynamoClient(ctx)
	if err != nil {
		return err
	}
	_, err = client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName:           &s.tableName,
		Key:                 userItemKey(userID),
		UpdateExpression:    ptrString("REMOVE #avatar_url SET #avatar_status = :avatar_status, #avatar_updated_at = :avatar_updated_at"),
		ConditionExpression: ptrString("attribute_exists(#pk)"),
		ExpressionAttributeNames: map[string]string{
			"#pk":                "pk",
			"#avatar_url":        "avatar_url",
			"#avatar_status":     "avatar_status",
			"#avatar_updated_at": "avatar_updated_at",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":avatar_status":     stringAttr("NONE"),
			":avatar_updated_at": stringAttr(formatInstant(now)),
		},
	})
	if err != nil {
		return fmt.Errorf("clear user avatar: %w", err)
	}
	return nil
}

// UpdateUserProfile sets the display name and bio on the user item.
func (s *dynamoMissionStore) UpdateUserProfile(ctx context.Context, userID string, displayName string, bio string, now time.Time) error {
	client, err := s.dynamoClient(ctx)
	if err != nil {
		return err
	}
	_, err = client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName:           &s.tableName,
		Key:                 userItemKey(userID),
		UpdateExpression:    ptrString("SET #display_name = :display_name, #bio = :bio"),
		ConditionExpression: ptrString("attribute_exists(#pk)"),
		ExpressionAttributeNames: map[string]string{
			"#pk":           "pk",
			"#display_name": "display_name",
			"#bio":          "bio",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":display_name": stringAttr(displayName),
			":bio":          stringAttr(bio),
		},
	})
	if err != nil {
		return fmt.Errorf("update user profile: %w", err)
	}
	return nil
}
