import { Button, Card, Select, Table, Tag, message } from 'antd'
import { useEffect, useMemo, useState } from 'react'
import { listMyApplications, listOrgApplications, updateApplicationStatus } from '@/api/application'
import { listMyHandovers, listOrgHandovers } from '@/api/handover'
import ApplicationStatusBadge from '@/components/common/ApplicationStatusBadge'
import HandoverPanel from '@/components/common/HandoverPanel'
import { HandoverStatusMap } from '@/constants/handover'
import { useAuth } from '@/hooks/useAuth'
import type { AdoptionApplication, HandoverAppointment } from '@/types/api'
import { formatDate, formatDateTime } from '@/utils/dateFormat'

export default function Applications() {
  const { isOrg } = useAuth()
  const [apps, setApps] = useState<AdoptionApplication[]>([])
  const [handovers, setHandovers] = useState<HandoverAppointment[]>([])
  const [status, setStatus] = useState('')

  async function load(s = status) {
    if (isOrg) {
      const [appList, handoverList] = await Promise.all([listOrgApplications(s), listOrgHandovers()])
      setApps(appList)
      setHandovers(handoverList)
    } else {
      const [appList, handoverList] = await Promise.all([listMyApplications(), listMyHandovers()])
      setApps(appList)
      setHandovers(handoverList)
    }
  }
  useEffect(() => {
    load()
  }, [isOrg])

  // Latest handover record of each application (records are ordered newest first).
  const handoverByApp = useMemo(() => {
    const map = new Map<number, HandoverAppointment>()
    handovers.forEach((h) => {
      if (!map.has(h.application_id)) map.set(h.application_id, h)
    })
    return map
  }, [handovers])

  async function changeStatus(id: number, next: string) {
    await updateApplicationStatus(id, next)
    message.success('状态已更新')
    await load()
  }

  return (
    <div>
      <h1>领养申请</h1>
      {isOrg && (
        <Select
          style={{ width: 180, marginBottom: 12 }}
          placeholder="按状态筛选"
          allowClear
          value={status || undefined}
          onChange={(v) => {
            setStatus(v || '')
            load(v || '')
          }}
          options={['submitted', 'org_review', 'communicating', 'confirmed', 'offline_interview', 'approved', 'rejected'].map((s) => ({ value: s, label: s }))}
        />
      )}
      <Table
        rowKey="id"
        dataSource={apps}
        pagination={false}
        expandable={{
          expandedRowRender: (r) => <HandoverPanel applicationId={r.id} approved={r.status === 'approved'} isOrg={isOrg} />,
          rowExpandable: (r) => isOrg || r.status === 'approved' || handoverByApp.has(r.id),
        }}
        columns={[
          { title: 'ID', dataIndex: 'id' },
          { title: '宠物 ID', dataIndex: 'pet_id' },
          { title: '申请时间', dataIndex: 'created_at', render: (v: string) => formatDate(v) },
          { title: '审核状态', dataIndex: 'status', render: (v: string) => <ApplicationStatusBadge status={v} /> },
          {
            title: '交接预约',
            render: (_, r) => {
              const h = handoverByApp.get(r.id)
              if (!h) return <Tag>未安排</Tag>
              const meta = HandoverStatusMap[h.status]
              return (
                <span>
                  <Tag color={meta.color}>{meta.text}</Tag>
                  {h.selected_slot && <span style={{ fontSize: 12 }}>{formatDateTime(h.selected_slot)}</span>}
                </span>
              )
            },
          },
          {
            title: '操作',
            render: (_, r) =>
              isOrg ? (
                <Button size="small" onClick={() => changeStatus(r.id, r.status === 'submitted' ? 'org_review' : 'approved')}>
                  推进审核
                </Button>
              ) : null,
          },
        ]}
      />
      {!apps.length && <Card style={{ marginTop: 12 }}>暂无申请记录</Card>}
    </div>
  )
}
