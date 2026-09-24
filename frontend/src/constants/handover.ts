import type { HandoverStatus } from '@/types/api'

export const HandoverStatusMap: Record<HandoverStatus, { text: string; color: string }> = {
  offered: { text: '待选时段', color: 'gold' },
  pending: { text: '待机构确认', color: 'processing' },
  confirmed: { text: '已确认', color: 'green' },
  cancelled: { text: '已取消', color: 'default' },
  expired: { text: '已失效', color: 'red' },
}

export const HandoverCancelledByText: Record<string, string> = {
  user: '领养人',
  org: '机构',
}
