package handlers

import "testing"

func nodeByID(t *testing.T, res achievementsResponse, id string) achievementNode {
	t.Helper()
	for _, branch := range res.Branches {
		for _, node := range branch.Nodes {
			if node.ID == id {
				return node
			}
		}
	}
	t.Fatalf("node %q not found in response", id)
	return achievementNode{}
}

func TestBuildAchievements(t *testing.T) {
	now := mustTime(t, "2026-06-04T12:00:00Z")

	tests := []struct {
		name           string
		records        stateRecords
		userID         string
		wantEarned     int
		wantEarnedByID map[string]bool
	}{
		{
			name:       "empty records earn nothing",
			records:    stateRecords{},
			userID:     "u1",
			wantEarned: 0,
			wantEarnedByID: map[string]bool{
				"MANIFEST_SIGNED": false,
				"THE_100":         false,
				"AIRLOCKED":       false,
			},
		},
		{
			name:   "fresh boarder earns all boarding",
			userID: "u1",
			records: stateRecords{
				Users: []airlockUser{{UserID: "u1", Handle: "maya", AvatarStatus: "READY"}},
				Seats: []airlockSeat{{SeatID: "SEAT-007", SeatNumber: 7, OccupantUserID: "u1", Status: "OCCUPIED", BoardedAt: "2026-06-03T12:00:00Z"}},
			},
			wantEarned: 3,
			wantEarnedByID: map[string]bool{
				"MANIFEST_SIGNED": true,
				"THE_100":         true,
				"SUIT_ON":         true,
				"FIRST_SHIP":      false,
				"DAYS_30":         false,
			},
		},
		{
			name:   "ten confirmed earns all shipping",
			userID: "u1",
			records: stateRecords{
				Users:    []airlockUser{{UserID: "u1", Handle: "maya"}},
				Seats:    []airlockSeat{{SeatID: "SEAT-007", SeatNumber: 7, OccupantUserID: "u1", Status: "OCCUPIED"}},
				Missions: confirmedMissions("SEAT-007", 10),
			},
			wantEarnedByID: map[string]bool{
				"FIRST_SHIP":     true,
				"REPEAT_SHIPPER": true,
				"VETERAN":        true,
				"PROLIFIC":       true,
				"UNBROKEN":       true,
			},
		},
		{
			name:   "streak resets on airlock in the middle",
			userID: "u1",
			records: stateRecords{
				Users: []airlockUser{{UserID: "u1", Handle: "maya"}},
				Seats: []airlockSeat{{SeatID: "SEAT-007", SeatNumber: 7, OccupantUserID: "u1", Status: "OCCUPIED"}},
				Missions: []airlockMission{
					{MissionID: "M1", SeatID: "SEAT-007", Status: "CONFIRMED", DeclaredAt: "2026-01-01T00:00:00Z"},
					{MissionID: "M2", SeatID: "SEAT-007", Status: "CONFIRMED", DeclaredAt: "2026-01-02T00:00:00Z"},
					{MissionID: "M3", SeatID: "SEAT-007", Status: "AIRLOCKED", DeclaredAt: "2026-01-03T00:00:00Z"},
					{MissionID: "M4", SeatID: "SEAT-007", Status: "CONFIRMED", DeclaredAt: "2026-01-04T00:00:00Z"},
					{MissionID: "M5", SeatID: "SEAT-007", Status: "CONFIRMED", DeclaredAt: "2026-01-05T00:00:00Z"},
					{MissionID: "M6", SeatID: "SEAT-007", Status: "CONFIRMED", DeclaredAt: "2026-01-06T00:00:00Z"},
					{MissionID: "M7", SeatID: "SEAT-007", Status: "CONFIRMED", DeclaredAt: "2026-01-07T00:00:00Z"},
					{MissionID: "M8", SeatID: "SEAT-007", Status: "CONFIRMED", DeclaredAt: "2026-01-08T00:00:00Z"},
				},
			},
			// longest run after the reset is 5 (M4..M8).
			wantEarnedByID: map[string]bool{
				"BACK_TO_BACK": true,
				"HOT_STREAK":   true,
				"UNBROKEN":     true,
				"AIRLOCKED":    true,
			},
		},
		{
			name:   "redemption all earned",
			userID: "u1",
			records: stateRecords{
				Users: []airlockUser{{UserID: "u1", Handle: "maya"}},
				Seats: []airlockSeat{
					{SeatID: "SEAT-007", SeatNumber: 7, OccupantUserID: "u1", Status: "VACATED", VacatedAt: "2026-02-01T00:00:00Z", ReboardedAsSeatID: "SEAT-101"},
					{SeatID: "SEAT-101", SeatNumber: 101, OccupantUserID: "u1", Status: "OCCUPIED", BoardedAt: "2026-02-01T00:00:00Z"},
				},
				Missions: []airlockMission{
					{MissionID: "M1", SeatID: "SEAT-007", Status: "AIRLOCKED", DeclaredAt: "2026-01-01T00:00:00Z", AirlockedAt: "2026-02-01T00:00:00Z", Redeemed: true},
				},
			},
			wantEarnedByID: map[string]bool{
				"AIRLOCKED":  true,
				"RE_BOARDED": true,
				"REDEEMED":   true,
			},
		},
		{
			name:   "long tenure earns all longevity",
			userID: "u1",
			records: stateRecords{
				Users: []airlockUser{{UserID: "u1", Handle: "maya"}},
				Seats: []airlockSeat{
					{SeatID: "SEAT-007", SeatNumber: 7, OccupantUserID: "u1", Status: "VACATED", BoardedAt: "2025-01-01T00:00:00Z", VacatedAt: "2025-07-01T00:00:00Z"},
					{SeatID: "SEAT-101", SeatNumber: 101, OccupantUserID: "u1", Status: "OCCUPIED", BoardedAt: "2025-07-01T00:00:00Z"},
				},
			},
			wantEarnedByID: map[string]bool{
				"DAYS_30":  true,
				"DAYS_100": true,
				"YEAR_ONE": true,
			},
		},
		{
			name:   "all in requires both flags on same stack",
			userID: "u1",
			records: stateRecords{
				Users: []airlockUser{{UserID: "u1", Handle: "maya"}},
				Seats: []airlockSeat{
					{SeatID: "SEAT-007", SeatNumber: 7, OccupantUserID: "u1", Status: "OCCUPIED"},
					{SeatID: "SEAT-008", SeatNumber: 8, OccupantUserID: "u1", Status: "OCCUPIED"},
				},
				Stacks: []airlockStack{
					{SeatID: "SEAT-007", CrewAlertEnabled: true},
					{SeatID: "SEAT-008", TheWakeEnabled: true},
				},
			},
			wantEarnedByID: map[string]bool{
				"CREW_ALERTED": true,
				"WAKE_ARMED":   true,
				"ALL_IN":       false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildAchievements(tt.records, tt.userID, now)
			if got.TotalCount != 19 {
				t.Fatalf("TotalCount = %d, want 19", got.TotalCount)
			}
			if tt.wantEarned != 0 || tt.name == "empty records earn nothing" {
				if got.EarnedCount != tt.wantEarned {
					t.Fatalf("EarnedCount = %d, want %d", got.EarnedCount, tt.wantEarned)
				}
			}
			for id, want := range tt.wantEarnedByID {
				if node := nodeByID(t, got, id); node.Earned != want {
					t.Fatalf("node %q earned = %v, want %v", id, node.Earned, want)
				}
			}
		})
	}
}

func TestBuildAchievementsAirlockedToneIsHatch(t *testing.T) {
	now := mustTime(t, "2026-06-04T12:00:00Z")
	got := buildAchievements(stateRecords{}, "u1", now)
	if node := nodeByID(t, got, "AIRLOCKED"); node.Tone != "hatch" {
		t.Fatalf("AIRLOCKED tone = %q, want hatch", node.Tone)
	}
	if node := nodeByID(t, got, "MANIFEST_SIGNED"); node.Tone != "ice" {
		t.Fatalf("MANIFEST_SIGNED tone = %q, want ice", node.Tone)
	}
}

func confirmedMissions(seatID string, count int) []airlockMission {
	missions := make([]airlockMission, 0, count)
	days := []string{
		"2026-01-01T00:00:00Z", "2026-01-02T00:00:00Z", "2026-01-03T00:00:00Z",
		"2026-01-04T00:00:00Z", "2026-01-05T00:00:00Z", "2026-01-06T00:00:00Z",
		"2026-01-07T00:00:00Z", "2026-01-08T00:00:00Z", "2026-01-09T00:00:00Z",
		"2026-01-10T00:00:00Z",
	}
	for i := 0; i < count; i++ {
		missions = append(missions, airlockMission{
			MissionID:  "M" + days[i][8:10],
			SeatID:     seatID,
			Status:     "CONFIRMED",
			DeclaredAt: days[i],
		})
	}
	return missions
}
