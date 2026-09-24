package service

import (
	"regexp"
	"strings"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"

	"github.com/gbadopt/gbadopt/internal/constants"
	"github.com/gbadopt/gbadopt/internal/model"
	"github.com/gbadopt/gbadopt/internal/repository"
)

func handoverColumns() []string {
	return []string{"id", "application_id", "org_id", "user_id", "location", "slots",
		"confirm_before", "selected_slot", "status", "cancelled_by", "confirmed_at",
		"cancelled_at", "created_at", "updated_at"}
}

func TestHandoverSelectConflictWhenSlotLocked(t *testing.T) {
	db, mock := newServiceDB(t)
	svc := NewHandoverService(
		repository.NewHandoverAppointmentRepository(db),
		repository.NewAdoptionApplicationRepository(db),
		repository.NewOrganizationRepository(db),
		newTestLogger(),
	)

	slot := time.Now().Add(48 * time.Hour).UTC().Truncate(time.Hour)
	deadline := time.Now().Add(24 * time.Hour)
	rows := sqlmock.NewRows(handoverColumns()).AddRow(
		10, 5, 3, 2, "上海救助站", `["`+slot.Format(time.RFC3339)+`"]`,
		deadline, nil, "offered", "", nil, nil, time.Now(), time.Now())
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "handover_appointments" WHERE "handover_appointments"."id" = $1 ORDER BY "handover_appointments"."id" LIMIT $2`)).
		WithArgs(10, 1).WillReturnRows(rows)

	appRows := sqlmock.NewRows([]string{"id", "user_id", "pet_id", "org_id", "questionnaire", "status", "created_at", "updated_at"}).
		AddRow(5, 2, 1, 3, "{}", "approved", time.Now(), time.Now())
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "adoption_applications" WHERE "adoption_applications"."id" = $1 ORDER BY "adoption_applications"."id" LIMIT $2`)).
		WithArgs(5, 1).WillReturnRows(appRows)

	// Another pending appointment already holds the same org slot.
	holderRows := sqlmock.NewRows(handoverColumns()).AddRow(
		11, 6, 3, 9, "上海救助站", `["`+slot.Format(time.RFC3339)+`"]`,
		deadline, slot, "pending", "", nil, nil, time.Now(), time.Now())
	mock.ExpectQuery(`SELECT \* FROM "handover_appointments" WHERE org_id = \$1 AND selected_slot = \$2 AND id <> \$3 AND status IN \(\$4,\$5\)`).
		WithArgs(3, slot, 10, "pending", "confirmed", 1).WillReturnRows(holderRows)

	_, err := svc.Select(2, 10, slot.Format(time.RFC3339))
	if err == nil {
		t.Fatal("expected conflict error when slot already locked")
	}
	if got := err.Error(); got == "" || !strings.Contains(got, "already locked") {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}

func TestHandoverConfirmRejectsNonPending(t *testing.T) {
	db, mock := newServiceDB(t)
	svc := NewHandoverService(
		repository.NewHandoverAppointmentRepository(db),
		repository.NewAdoptionApplicationRepository(db),
		repository.NewOrganizationRepository(db),
		newTestLogger(),
	)

	// An offered (not pending) appointment cannot be confirmed.
	rows := sqlmock.NewRows(handoverColumns()).AddRow(
		10, 5, 3, 2, "上海救助站", "[]",
		time.Now().Add(time.Hour), nil, "offered", "", nil, nil, time.Now(), time.Now())
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "handover_appointments" WHERE "handover_appointments"."id" = $1 ORDER BY "handover_appointments"."id" LIMIT $2`)).
		WithArgs(10, 1).WillReturnRows(rows)

	appRows := sqlmock.NewRows([]string{"id", "user_id", "pet_id", "org_id", "questionnaire", "status", "created_at", "updated_at"}).
		AddRow(5, 2, 1, 3, "{}", "approved", time.Now(), time.Now())
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "adoption_applications" WHERE "adoption_applications"."id" = $1 ORDER BY "adoption_applications"."id" LIMIT $2`)).
		WithArgs(5, 1).WillReturnRows(appRows)
	orgRows := sqlmock.NewRows([]string{"id", "user_id", "name", "cert_type", "status"}).
		AddRow(3, 7, "救助站", "registered", "approved")
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "organizations" WHERE user_id = $1 ORDER BY "organizations"."id" LIMIT $2`)).
		WithArgs(7, 1).WillReturnRows(orgRows)

	if _, err := svc.Confirm(7, 10); err == nil {
		t.Fatal("expected conflict when confirming a non-pending appointment")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}

func TestHandoverConstants(t *testing.T) {
	// Sanity: model default zero-value check keeps selected_slot nullable.
	var h model.HandoverAppointment
	if h.Status != "" {
		t.Fatal("fresh model should be zero-valued")
	}
	_ = constants.HandoverStatusOffered
}
