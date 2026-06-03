package handlers

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	ErrBoardingInvalidHandle = errors.New("boarding handle is invalid")
	ErrBoardingAlreadySeated = errors.New("user already has an occupied seat")
	ErrBoardingHandleTaken   = errors.New("handle already belongs to another user")
	ErrBoardingPlanUnderpaid = errors.New("purchased plan does not cover this seat's price")
)

var handlePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{1,30}[a-z0-9]$`)

const (
	firstPost100SeatNumber    = 101
	boardingDisplayPriceCents = 4200
)

type boardingFulfillment struct {
	User               airlockUser
	Seat               airlockSeat
	Event              airlockEvent
	LinkedVacatedSeats []airlockSeat
}

func buildBoarding(records stateRecords, userID string, email string, handle string, displayName string, planAmountCents int, now time.Time) (boardingFulfillment, error) {
	handle, err := normalizeHandle(handle, email, userID)
	if err != nil {
		return boardingFulfillment{}, err
	}
	displayName = strings.Join(strings.Fields(strings.TrimSpace(displayName)), " ")
	if len(displayName) > 80 {
		displayName = displayName[:80]
	}
	if displayName == "" {
		displayName = handle
	}

	for _, seat := range records.Seats {
		if seat.OccupantUserID == userID && normalizeStatus(seat.Status) == "OCCUPIED" && seat.VacatedAt == "" {
			return boardingFulfillment{}, ErrBoardingAlreadySeated
		}
	}
	for _, user := range records.Users {
		if strings.EqualFold(user.Handle, handle) && user.UserID != userID {
			return boardingFulfillment{}, ErrBoardingHandleTaken
		}
	}

	seatNumber := nextAssignableSeatNumber(records, now)
	tierPaid := tierPaidForSeat(seatNumber)
	reboard := userHasVacatedSeat(records, userID)
	if reboard {
		seatNumber = nextPost100SeatNumber(records, now)
		tierPaid = 5
	}

	if planAmountCents < priceForSeat(seatNumber).Cents {
		return boardingFulfillment{}, ErrBoardingPlanUnderpaid
	}

	nowISO := formatInstant(now)
	seatID := seatIDForNumber(seatNumber)
	seatLabelValue := seatLabel(seatNumber)

	user := airlockUser{
		UserID:      userID,
		Handle:      handle,
		DisplayName: displayName,
		Email:       email,
		AvatarSeed:  handle,
	}
	seat := airlockSeat{
		SeatID:         seatID,
		SeatNumber:     seatNumber,
		Cohort:         cohortForSeat(seatNumber, ""),
		OccupantUserID: userID,
		Status:         "OCCUPIED",
		TierPaid:       tierPaid,
		PricePaidCents: boardingDisplayPriceCents,
		BoardedAt:      nowISO,
	}
	eventKind := "BOARDED"
	eventAction := "boarded"
	if reboard {
		eventKind = "RE_BOARDED"
		eventAction = "re-boarded"
	}
	event := airlockEvent{
		EventID:    eventIDForBoarding(seatID),
		Kind:       eventKind,
		UserID:     userID,
		SeatID:     seatID,
		OccurredAt: nowISO,
		Message:    "@" + handle + " " + eventAction + " THE AIRLOCK - seat " + seatLabelValue,
	}
	return boardingFulfillment{
		User:               user,
		Seat:               seat,
		Event:              event,
		LinkedVacatedSeats: vacatedSeatsToLinkForReboard(records, userID, seatID, seatLabelValue),
	}, nil
}

func vacatedSeatsToLinkForReboard(records stateRecords, userID string, reboardedSeatID string, reboardedSeatLabel string) []airlockSeat {
	var latest airlockSeat
	hasLatest := false
	for _, seat := range records.Seats {
		if seat.OccupantUserID != userID {
			continue
		}
		if normalizeStatus(seat.Status) != "VACATED" && seat.VacatedAt == "" {
			continue
		}
		if seat.ReboardedAsSeatID != "" {
			continue
		}
		if !hasLatest || seat.VacatedAt > latest.VacatedAt || (seat.VacatedAt == latest.VacatedAt && seat.SeatNumber > latest.SeatNumber) {
			latest = seat
			hasLatest = true
		}
	}
	if !hasLatest {
		return nil
	}
	latest.ReboardedAsSeatID = reboardedSeatID
	latest.ReboardedAsSeatLabel = reboardedSeatLabel
	return []airlockSeat{latest}
}

func userHasVacatedSeat(records stateRecords, userID string) bool {
	for _, seat := range records.Seats {
		if seat.OccupantUserID != userID {
			continue
		}
		if normalizeStatus(seat.Status) == "VACATED" || seat.VacatedAt != "" {
			return true
		}
	}
	return false
}

func normalizeHandle(handle string, email string, userID string) (string, error) {
	handle = strings.ToLower(strings.TrimSpace(handle))
	if handle == "" {
		source := email
		if source == "" {
			source = userID
		}
		if at := strings.Index(source, "@"); at > 0 {
			source = source[:at]
		}
		handle = strings.ToLower(source)
	}
	replacer := strings.NewReplacer(".", "-", " ", "-", "+", "-", "@", "-")
	handle = replacer.Replace(handle)
	handle = strings.Trim(handle, "-_")
	if !handlePattern.MatchString(handle) {
		return "", ErrBoardingInvalidHandle
	}
	return handle, nil
}

func nextAssignableSeatNumber(records stateRecords, now time.Time) int {
	maxOriginalSeatNumber := 0
	for _, seat := range records.Seats {
		if seat.SeatNumber > 100 {
			continue
		}
		if seat.SeatNumber > maxOriginalSeatNumber {
			maxOriginalSeatNumber = seat.SeatNumber
		}
	}
	if maxOriginalSeatNumber < 100 {
		return maxOriginalSeatNumber + 1
	}
	return nextPost100SeatNumber(records, now)
}

func nextPost100SeatNumber(records stateRecords, now time.Time) int {
	maxSeatNumber := firstPost100SeatNumber - 1
	for _, seat := range records.Seats {
		if seat.SeatNumber > maxSeatNumber {
			maxSeatNumber = seat.SeatNumber
		}
	}
	return maxSeatNumber + 1
}

var planAmountCentsBySlug = map[string]int{
	"tier-01": 4200,
	"tier-02": 8400,
	"tier-03": 16800,
	"tier-04": 33600,
}

// planAmountCentsForSlug returns the amount in cents the given Clerk plan slug
// entitles, or 0 for an unknown/legacy slug (fail-closed: 0 never satisfies a
// ladder price). Mirrors infra/clerk/billing.json plan.amount and priceForTier.
func planAmountCentsForSlug(slug string) int {
	return planAmountCentsBySlug[slug]
}

func tierPaidForSeat(seatNumber int) int {
	tier := tierForSeat(seatNumber).ID
	if tier == "∞" {
		return 5
	}
	value, err := strconv.Atoi(tier)
	if err != nil {
		return 5
	}
	return value
}

func seatIDForNumber(seatNumber int) string {
	return fmt.Sprintf("SEAT-%03d", seatNumber)
}

func eventIDForBoarding(seatID string) string {
	return "EVENT-BOARDED-" + seatID
}
