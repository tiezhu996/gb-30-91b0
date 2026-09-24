package service

import (
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gbadopt/gbadopt/internal/constants"
	"github.com/gbadopt/gbadopt/internal/dto"
	"github.com/gbadopt/gbadopt/internal/model"
	"github.com/gbadopt/gbadopt/internal/repository"
)

func TestHandoverCreateOfferValidation(t *testing.T) {
	db, _ := newServiceDB(t)
	svc := NewHandoverService(
		db,
		repository.NewHandoverOfferRepository(db),
		repository.NewHandoverAppointmentRepository(db),
		nil, nil, newTestLogger())

	base := func() dto.HandoverOfferCreateRequest {
		return dto.HandoverOfferCreateRequest{
			ApplicationID: 1,
			Location:      "救助站前台",
			Deadline:      time.Now().Add(48 * time.Hour),
			Slots: []dto.HandoverSlot{{
				StartAt: time.Now().Add(72 * time.Hour),
				EndAt:   time.Now().Add(73 * time.Hour),
			}},
		}
	}

	t.Run("deadline in the past", func(t *testing.T) {
		req := base()
		req.Deadline = time.Now().Add(-time.Hour)
		if _, err := svc.CreateOffer(1, req); err == nil {
			t.Fatal("expected error for past deadline")
		}
	})

	t.Run("empty location", func(t *testing.T) {
		req := base()
		req.Location = ""
		if _, err := svc.CreateOffer(1, req); err == nil {
			t.Fatal("expected error for empty location")
		}
	})

	t.Run("slot end before start", func(t *testing.T) {
		req := base()
		req.Slots[0].EndAt = req.Slots[0].StartAt.Add(-time.Hour)
		if _, err := svc.CreateOffer(1, req); err == nil {
			t.Fatal("expected error for invalid slot window")
		}
	})

	t.Run("slot in the past", func(t *testing.T) {
		req := base()
		req.Slots[0].StartAt = time.Now().Add(-time.Hour)
		req.Slots[0].EndAt = time.Now().Add(time.Hour)
		if _, err := svc.CreateOffer(1, req); err == nil {
			t.Fatal("expected error for past slot")
		}
	})

	t.Run("duplicated slots", func(t *testing.T) {
		req := base()
		req.Slots = append(req.Slots, req.Slots[0])
		if _, err := svc.CreateOffer(1, req); err == nil {
			t.Fatal("expected error for duplicated slots")
		}
	})
}

func TestHandoverOfferHasSlot(t *testing.T) {
	db, _ := newServiceDB(t)
	svc := NewHandoverService(db, nil, nil, nil, nil, newTestLogger())

	start := time.Date(2026, 10, 1, 10, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)
	offer := &model.HandoverOffer{
		Slots: `[{"start_at":"2026-10-01T10:00:00Z","end_at":"2026-10-01T11:00:00Z"}]`,
	}
	if !svc.offerHasSlot(offer, start, end) {
		t.Fatal("exact slot should match")
	}
	if svc.offerHasSlot(offer, start, start.Add(90*time.Minute)) {
		t.Fatal("different end should not match")
	}
	if svc.offerHasSlot(&model.HandoverOffer{Slots: "not-json"}, start, end) {
		t.Fatal("malformed slots should never match")
	}
}

func TestHandoverStatusGuards(t *testing.T) {
	if !constants.IsActiveHandoverAppointmentStatus(constants.HandoverApptPending) {
		t.Fatal("pending occupies a slot")
	}
	if !constants.IsActiveHandoverAppointmentStatus(constants.HandoverApptConfirmed) {
		t.Fatal("confirmed occupies a slot")
	}
	for _, s := range []string{constants.HandoverApptCancelled, constants.HandoverApptExpired} {
		if constants.IsActiveHandoverAppointmentStatus(s) {
			t.Fatalf("%s must release the slot", s)
		}
	}
	if constants.IsValidHandoverAppointmentStatus("bogus") {
		t.Fatal("unknown status must be invalid")
	}
}

// TestMarkExpiredUsesPendingFilter documents the sweep semantics with a mocked UPDATE.
func TestMarkExpiredUsesPendingFilter(t *testing.T) {
	db, mock := newServiceDB(t)
	svc := NewHandoverService(db, repository.NewHandoverOfferRepository(db), repository.NewHandoverAppointmentRepository(db), nil, nil, newTestLogger())

	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE "handover_appointments" SET "status"=\$1,"updated_at"=\$2 WHERE status = \$3 AND deadline < \$4`).
		WithArgs(constants.HandoverApptExpired, sqlmock.AnyArg(), constants.HandoverApptPending, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectCommit()

	n, err := svc.apptRepo.MarkExpired(time.Now())
	if err != nil {
		t.Fatalf("MarkExpired: %v", err)
	}
	if n != 2 {
		t.Fatalf("expected 2 expired rows, got %d", n)
	}
}
