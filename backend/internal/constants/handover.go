package constants

// Handover appointment status values.
const (
	HandoverStatusOffered   = "offered"   // 机构给出地点/截止时间/可预约时段，等待领养人选择
	HandoverStatusPending   = "pending"   // 领养人已选中时段并占用，等待机构确认
	HandoverStatusConfirmed = "confirmed" // 机构已确认，不可再改
	HandoverStatusCancelled = "cancelled" // 确认前任一方取消，时段已释放
	HandoverStatusExpired   = "expired"   // 超过截止时间未确认，时段已释放
)

// HandoverCancelledBy values.
const (
	HandoverCancelByUser = "user"
	HandoverCancelByOrg  = "org"
)

// Active handover statuses still occupy a slot (offered occupies none;
// pending/confirmed lock the selected slot).
var handoverLockingStatuses = []string{HandoverStatusPending, HandoverStatusConfirmed}

// ValidHandoverStatuses returns all accepted handover appointment statuses.
func ValidHandoverStatuses() []string {
	return []string{
		HandoverStatusOffered, HandoverStatusPending, HandoverStatusConfirmed,
		HandoverStatusCancelled, HandoverStatusExpired,
	}
}

// IsValidHandoverStatus reports whether a status is known.
func IsValidHandoverStatus(s string) bool {
	for _, v := range ValidHandoverStatuses() {
		if v == s {
			return true
		}
	}
	return false
}

// HandoverLockingStatuses returns the statuses whose selected slot is occupied.
func HandoverLockingStatuses() []string {
	out := make([]string, len(handoverLockingStatuses))
	copy(out, handoverLockingStatuses)
	return out
}

// IsHandoverTerminal reports whether a handover can no longer change.
func IsHandoverTerminal(s string) bool {
	return s == HandoverStatusConfirmed || s == HandoverStatusCancelled || s == HandoverStatusExpired
}
