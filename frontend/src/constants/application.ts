export type ApplicationStatus =
  | 'submitted'
  | 'org_review'
  | 'communicating'
  | 'confirmed'
  | 'offline_interview'
  | 'approved'
  | 'rejected'

export const ApplicationStatusMap: Record<ApplicationStatus, { text: string; color: string }> = {
  submitted: { text: '已提交', color: 'blue' },
  org_review: { text: '机构审核中', color: 'gold' },
  communicating: { text: '沟通中', color: 'cyan' },
  confirmed: { text: '已确认', color: 'geekblue' },
  offline_interview: { text: '线下面签', color: 'purple' },
  approved: { text: '已通过', color: 'green' },
  rejected: { text: '已拒绝', color: 'red' },
}

export type HandoverApptStatus = 'pending' | 'confirmed' | 'cancelled' | 'expired'

export const HandoverApptStatusMap: Record<HandoverApptStatus, { text: string; color: string }> = {
  pending: { text: '待机构确认', color: 'gold' },
  confirmed: { text: '已确认', color: 'green' },
  cancelled: { text: '已取消', color: 'default' },
  expired: { text: '已失效', color: 'red' },
}

export type HandoverSlotStatus = 'free' | 'locked_by_me' | 'occupied'

export const HandoverSlotStatusMap: Record<HandoverSlotStatus, { text: string; color: string }> = {
  free: { text: '可预约', color: 'green' },
  locked_by_me: { text: '我已锁定', color: 'blue' },
  occupied: { text: '已被其他申请占用', color: 'red' },
}
