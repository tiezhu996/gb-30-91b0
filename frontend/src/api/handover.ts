import request from '@/utils/request'
import type {
  HandoverAppointment,
  HandoverOffer,
  HandoverSlot,
} from '@/types/api'

// 机构按申请发起交接方案
export function createHandoverOffer(payload: {
  application_id: number
  location: string
  deadline: string
  slots: HandoverSlot[]
}) {
  return request.post<never, HandoverOffer>('/handovers/offers', payload)
}

// 查看申请的最新交接方案（时段占用情况 + 当前预约）
export function getHandoverOffer(applicationId: number) {
  return request.get<never, HandoverOffer | null>(`/applications/${applicationId}/handover`)
}

// 领养人选中时段并占用
export function lockHandoverSlot(payload: { offer_id: number; slot: HandoverSlot }) {
  return request.post<never, HandoverAppointment>('/handovers/appointments', payload)
}

// 确认前机构/领养人取消，释放时段
export function cancelHandoverAppointment(id: number) {
  return request.put<never, HandoverAppointment>(`/handovers/appointments/${id}/cancel`)
}

// 机构确认预约
export function confirmHandoverAppointment(id: number) {
  return request.put<never, HandoverAppointment>(`/handovers/appointments/${id}/confirm`)
}
