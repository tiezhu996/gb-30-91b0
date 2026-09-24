package constants

// HandoverAppointmentStatus enumerates the handover appointment lifecycle.
const (
	HandoverApptPending   = "pending"   // 时段已被领养人锁定，等待机构确认
	HandoverApptConfirmed = "confirmed" // 机构已确认
	HandoverApptCancelled = "cancelled" // 确认前由机构或领养人取消，时段已释放
	HandoverApptExpired   = "expired"   // 超过确认截止时间仍未确认，已失效
)

// ActiveHandoverAppointmentStatuses are statuses that still occupy a slot.
func ActiveHandoverAppointmentStatuses() []string {
	return []string{HandoverApptPending, HandoverApptConfirmed}
}

// IsActiveHandoverAppointmentStatus reports whether an appointment occupies a slot.
func IsActiveHandoverAppointmentStatus(s string) bool {
	return s == HandoverApptPending || s == HandoverApptConfirmed
}

// IsValidHandoverAppointmentStatus reports whether a status is known.
func IsValidHandoverAppointmentStatus(s string) bool {
	switch s {
	case HandoverApptPending, HandoverApptConfirmed, HandoverApptCancelled, HandoverApptExpired:
		return true
	default:
		return false
	}
}
