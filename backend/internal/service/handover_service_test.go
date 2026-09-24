package service

import (
	"testing"
	"time"

	"github.com/gbadopt/gbadopt/internal/constants"
)

func TestParseHandoverSlots(t *testing.T) {
	t1 := time.Now().Add(48 * time.Hour).Format(time.RFC3339)
	t2 := time.Now().Add(72 * time.Hour).Format(time.RFC3339)

	slots, err := parseHandoverSlots([]string{t1, t2, t1})
	if err != nil {
		t.Fatalf("parseHandoverSlots: %v", err)
	}
	if len(slots) != 2 {
		t.Fatalf("got %d slots, want 2 (dedup)", len(slots))
	}

	if _, err := parseHandoverSlots(nil); err == nil {
		t.Error("expected error for empty slots")
	}
	if _, err := parseHandoverSlots([]string{"not-a-time"}); err == nil {
		t.Error("expected error for invalid timestamp")
	}
}

func TestHandoverSlotMembership(t *testing.T) {
	slot := time.Now().Add(48 * time.Hour).Truncate(time.Hour)
	raw := `["` + slot.Format(time.RFC3339) + `"]`
	if !containsHandoverSlot(raw, slot) {
		t.Error("expected slot to be contained")
	}
	if containsHandoverSlot(raw, slot.Add(time.Hour)) {
		t.Error("did not expect a different slot to be contained")
	}
	if containsHandoverSlot("not-json", slot) {
		t.Error("invalid json must not contain anything")
	}
	if got := decodeHandoverSlots(""); got != nil {
		t.Errorf("empty raw should decode to nil, got %v", got)
	}
}

func TestHandoverStatusGuards(t *testing.T) {
	if !constants.IsHandoverTerminal(constants.HandoverStatusConfirmed) {
		t.Error("confirmed must be terminal (immutable)")
	}
	if !constants.IsHandoverTerminal(constants.HandoverStatusExpired) {
		t.Error("expired must be terminal (immutable)")
	}
	if !constants.IsHandoverTerminal(constants.HandoverStatusCancelled) {
		t.Error("cancelled must be terminal")
	}
	if constants.IsHandoverTerminal(constants.HandoverStatusPending) {
		t.Error("pending must not be terminal")
	}
	locking := constants.HandoverLockingStatuses()
	if len(locking) != 2 {
		t.Fatalf("locking statuses = %v, want pending/confirmed", locking)
	}
}
