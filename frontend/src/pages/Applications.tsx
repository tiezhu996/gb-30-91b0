import { Button, Select, Table, Tag, message } from 'antd'
import { useEffect, useState } from 'react'
import { listMyApplications, updateApplicationStatus } from '@/api/application'
import ApplicationStatusBadge from '@/components/common/ApplicationStatusBadge'
import HandoverPanel from '@/components/common/HandoverPanel'
import { HandoverApptStatusMap } from '@/constants/application'
import { useAuth } from '@/hooks/useAuth'
import type { AdoptionApplication } from '@/types/api'
import { formatDate } from '@/utils/dateFormat'

function HandoverBriefTag({ app }: { app: AdoptionApplication }) {
  if (app.status !== 'approved') return null
  const a = app.handover?.appointment
  if (a) {
    const meta = HandoverApptStatusMap[a.status]
    return <Tag color={meta.color}>{meta.text}</Tag>
  }
  if (app.handover) {
    return <Tag color="blue">待选时段</Tag>
  }
  return <Tag>待安排交接</Tag>
}

export default function Applications() {
  const { isOrg } = useAuth()
  const [apps, setApps] = useState<AdoptionApplication[]>([])
  const [status, setStatus] = useState('')

  async function load(s = status) {
    if (isOrg) {
      const res = await import('@/api/application').then((m) => m.listOrgApplications(s))
      setApps(res)
    } else {
      setApps(await listMyApplications())
    }
  }
  useEffect(() => {
    load()
  }, [isOrg])

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
      <Table<AdoptionApplication>
        rowKey="id"
        dataSource={apps}
        pagination={false}
        expandable={{
          rowExpandable: (r) => r.status === 'approved',
          expandedRowRender: (r) => <HandoverPanel application={r} isOrg={!!isOrg} onChanged={load} />,
        }}
        columns={[
          { title: 'ID', dataIndex: 'id' },
          { title: '宠物 ID', dataIndex: 'pet_id' },
          { title: '申请时间', dataIndex: 'created_at', render: (v: string) => formatDate(v) },
          { title: '状态', dataIndex: 'status', render: (v: string) => <ApplicationStatusBadge status={v} /> },
          {
            title: '交接预约',
            render: (_, r) => <HandoverBriefTag app={r} />,
          },
          {
            title: '操作',
            render: (_, r) =>
              isOrg ? (
                <Button
                  size="small"
                  disabled={r.status === 'approved' || r.status === 'rejected'}
                  onClick={() => changeStatus(r.id, r.status === 'submitted' ? 'org_review' : 'approved')}
                >
                  推进审核
                </Button>
              ) : null,
          },
        ]}
      />
      {!apps.length && <div style={{ marginTop: 12, color: '#999' }}>暂无申请记录</div>}
    </div>
  )
}
