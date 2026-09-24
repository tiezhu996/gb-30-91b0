import request from '@/utils/request'
import type { HandoverAppointment } from '@/types/api'

export interface HandoverOfferPayload {
  location: string
  slots: string[]
  confirm_before: string
}

// GET /applications/:id/handovers — full history for one application
export function listApplicationHandovers(applicationId: number) {
  return request.get<never, HandoverAppointment[]>(`/applications/${applicationId}/handovers`)
}

// GET /handovers/me — adopter's handover appointments
export function listMyHandovers() {
  return request.get<never, HandoverAppointment[]>('/handovers/me')
}

// GET /handovers/org — org's handover appointments
export function listOrgHandovers() {
  return request.get<never, HandoverAppointment[]>('/handovers/org')
}

// POST /applications/:id/handovers — org offers location/deadline/slots
export function offerHandover(applicationId: number, payload: HandoverOfferPayload) {
  return request.post<never, HandoverAppointment>(`/applications/${applicationId}/handovers`, payload)
}

// POST /handovers/:id/select — adopter occupies a slot
export function selectHandoverSlot(id: number, slot: string) {
  return request.post<never, HandoverAppointment>(`/handovers/${id}/select`, { slot })
}

// POST /handovers/:id/confirm — org confirms
export function confirmHandover(id: number) {
  return request.post<never, HandoverAppointment>(`/handovers/${id}/confirm`)
}

// POST /handovers/:id/cancel — both parties may cancel before confirmation
export function cancelHandover(id: number) {
  return request.post<never, HandoverAppointment>(`/handovers/${id}/cancel`)
}
