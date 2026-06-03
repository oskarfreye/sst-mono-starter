package handlers

import (
	"context"
	"fmt"
)

func (s *dynamoMissionStore) LoadStateRecords(ctx context.Context) (stateRecords, error) {
	client, err := s.dynamoClient(ctx)
	if err != nil {
		return stateRecords{}, err
	}
	return s.loadRecords(ctx, client)
}

type unavailableAchievementsStore struct{}

func (unavailableAchievementsStore) LoadStateRecords(context.Context) (stateRecords, error) {
	return stateRecords{}, fmt.Errorf("achievements store unavailable")
}
