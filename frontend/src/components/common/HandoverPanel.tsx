import { Button, Card, DatePicker, Divider, Empty, Form, Input, Modal, Popconfirm, Space, Table, Tag, TimePicker, message } from 'antd'
import { PlusOutlined, DeleteOutlined } from '@ant-design/icons'
import dayjs, { type Dayjs } from 'dayjs'
import { useMemo, useState } from 'react'
import {
  cancelHandoverAppointment,
  confirmHandoverAppointment,
  createHandoverOffer,
  lockHandoverSlot,
} from '@/api/handover'
import { HandoverApptStatusMap, HandoverSlotStatusMap } from '@/constants/application'
import type { AdoptionApplication, HandoverAppointment, HandoverSlotView } from '@/types/api'
import { formatDateTime } from '@/utils/dateFormat'

const { RangePicker: TimeRangePicker } = TimePicker

interface Props {
  application: AdoptionApplication
  isOrg: boolean
  onChanged: () => void | Promise<void>
}

const ACTIVE_STATUSES = ['pending', 'confirmed']
const TERMINAL_STATUSES = ['cancelled', 'expired']

function isActive(a?: HandoverAppointment | null) {
  return !!a && ACTIVE_STATUSES.includes(a.status)
}

export default function HandoverPanel({ application, isOrg, onChanged }: Props) {
  const offer = application.handover
  const appointment = offer?.appointment
  const [createOpen, setCreateOpen] = useState(false)
  const [busy, setBusy] = useState(false)

  const deadlinePassed = offer ? dayjs().isAfter(dayjs(offer.deadline)) : false
  const canOrgSchedule = isOrg && application.status === 'approved' && !isActive(appointment)
  const oldRecords = useMemo(
    () =>
      (application.history || []).filter(
        (h) => TERMINAL_STATUSES.includes(h.status) && (!appointment || h.id !== appointment.id),
      ),
    [application.history, appointment],
  )

  async function run(fn: () => Promise<unknown>, okText: string) {
    setBusy(true)
    try {
      await fn()
      message.success(okText)
      await onChanged()
    } finally {
      setBusy(false)
    }
  }

  if (application.status !== 'approved') return null

  const columns = [
    {
      title: '时段',
      render: (_: unknown, r: HandoverSlotView) => `${formatDateTime(r.start_at)} ~ ${formatDateTime(r.end_at)}`,
    },
    {
      title: '状态',
      render: (_: unknown, r: HandoverSlotView) => {
        const meta = HandoverSlotStatusMap[r.status]
        return <Tag color={meta.color}>{meta.text}</Tag>
      },
    },
    {
      title: '操作',
      render: (_: unknown, r: HandoverSlotView) => {
        if (isOrg) {
          if (r.status === 'occupied' || r.status === 'locked_by_me') {
            return r.application_id ? <Tag>申请 #{r.application_id}</Tag> : <Tag>已占用</Tag>
          }
          return <span style={{ color: '#999' }}>待领养人预约</span>
        }
        if (r.status === 'free' && !isActive(appointment) && !deadlinePassed) {
          return (
            <Button
              type="link"
              size="small"
              loading={busy}
              onClick={() =>
                run(
                  () => lockHandoverSlot({ offer_id: offer!.id, slot: { start_at: r.start_at, end_at: r.end_at } }),
                  '时段已锁定，等待机构确认',
                )
              }
            >
              预约此时段
            </Button>
          )
        }
        if (r.status === 'locked_by_me' && appointment?.status === 'pending') {
          return <Tag color="gold">等待机构确认中</Tag>
        }
        return null
      },
    },
  ]

  return (
    <Card size="small" style={{ marginTop: 12 }} title="🤝 交接预约">
      {/* 当前预约状态 */}
      {appointment && isActive(appointment) && (
        <Card size="small" style={{ marginBottom: 12, background: '#f6ffed' }}>
          <Space wrap>
            <Tag color={HandoverApptStatusMap[appointment.status].color}>
              {HandoverApptStatusMap[appointment.status].text}
            </Tag>
            <span>
              <b>{formatDateTime(appointment.start_at)}</b> ~ {formatDateTime(appointment.end_at)}
            </span>
            <span>地点：{appointment.location}</span>
            <span>确认截止：{formatDateTime(appointment.deadline)}</span>
            {appointment.confirmed_at && <span>确认时间：{formatDateTime(appointment.confirmed_at)}</span>}
          </Space>
          {appointment.status === 'pending' && (
            <Space style={{ marginTop: 8 }}>
              {isOrg && (
                <Popconfirm
                  title="确认该交接预约？"
                  description="确认后双方都不能再修改或取消。"
                  onConfirm={() => run(() => confirmHandoverAppointment(appointment.id), '预约已确认')}
                >
                  <Button type="primary" size="small" loading={busy}>
                    确认预约
                  </Button>
                </Popconfirm>
              )}
              <Popconfirm
                title="取消该预约？"
                description="取消后时段立即释放，可重新安排。"
                onConfirm={() => run(() => cancelHandoverAppointment(appointment.id), '预约已取消，时段已释放')}
              >
                <Button danger size="small" loading={busy}>
                  取消预约
                </Button>
              </Popconfirm>
            </Space>
          )}
        </Card>
      )}

      {/* 机构操作入口 */}
      {canOrgSchedule && (
        <Space style={{ marginBottom: offer ? 12 : 0 }}>
          <Button type="primary" icon={<PlusOutlined />} onClick={() => setCreateOpen(true)}>
            {offer ? '重新安排交接' : '发起交接预约'}
          </Button>
          {offer && deadlinePassed && !isActive(appointment) && (
            <Tag color="red">上个方案已过确认截止时间，可重新安排（旧记录已保留）</Tag>
          )}
          {offer && !deadlinePassed && !isActive(appointment) && (
            <Tag color="default">上个预约已取消，可重新安排（旧记录已保留）</Tag>
          )}
        </Space>
      )}

      {/* 方案与时段 */}
      {offer ? (
        <>
          <Space wrap style={{ marginBottom: 8 }}>
            <span>交接地点：<b>{offer.location}</b></span>
            <span>确认截止：{formatDateTime(offer.deadline)}</span>
            {deadlinePassed && !isActive(appointment) && <Tag color="red">已过截止时间</Tag>}
          </Space>
          <Table
            rowKey={(r) => `${r.start_at}-${r.end_at}`}
            size="small"
            pagination={false}
            dataSource={offer.slots}
            columns={columns}
          />
          {!isOrg && !offer.appointment && (
            <p style={{ color: '#999', marginTop: 8 }}>
              {deadlinePassed
                ? '确认截止时间已过，本次安排已失效，请等待机构重新安排。'
                : '请选择一个可预约时段，选中后机构只能与你确认该时段。'}
            </p>
          )}
        </>
      ) : (
        <Empty
          image={Empty.PRESENTED_IMAGE_SIMPLE}
          description={isOrg ? '申请通过后可发起交接预约' : '等待机构安排交接地点与可预约时段'}
        />
      )}

      {/* 历史记录：失效/取消的旧预约全部保留 */}
      {oldRecords.length > 0 && (
        <>
          <Divider style={{ margin: '12px 0' }}>历史安排</Divider>
          <Table
            rowKey="id"
            size="small"
            pagination={false}
            dataSource={oldRecords}
            columns={[
              { title: '时段', render: (_, r) => `${formatDateTime(r.start_at)} ~ ${formatDateTime(r.end_at)}` },
              { title: '地点', dataIndex: 'location' },
              {
                title: '结果',
                render: (_, r) => {
                  const meta = HandoverApptStatusMap[r.status]
                  return <Tag color={meta.color}>{meta.text}</Tag>
                },
              },
              { title: '创建时间', render: (_, r) => formatDateTime(r.created_at) },
            ]}
          />
        </>
      )}

      <CreateOfferModal
        open={createOpen}
        submitting={busy}
        onClose={() => setCreateOpen(false)}
        onSubmit={async (values) => {
          await run(async () => {
            await createHandoverOffer(values)
            setCreateOpen(false)
          }, '交接方案已发布')
        }}
        applicationId={application.id}
      />
    </Card>
  )
}

interface OfferFormValues {
  location: string
  deadline: Dayjs
  slots: { date: Dayjs; range: [Dayjs, Dayjs] }[]
}

function CreateOfferModal({
  open,
  submitting,
  applicationId,
  onClose,
  onSubmit,
}: {
  open: boolean
  submitting: boolean
  applicationId: number
  onClose: () => void
  onSubmit: (values: {
    application_id: number
    location: string
    deadline: string
    slots: { start_at: string; end_at: string }[]
  }) => Promise<void>
}) {
  const [form] = Form.useForm<OfferFormValues>()

  function handleOk() {
    form.validateFields().then((values) => {
      const slots = values.slots.map((s) => {
        const date = s.date.startOf('day')
        return {
          start_at: date.hour(s.range[0].hour()).minute(s.range[0].minute()).second(0).millisecond(0).toISOString(),
          end_at: date.hour(s.range[1].hour()).minute(s.range[1].minute()).second(0).millisecond(0).toISOString(),
        }
      })
      onSubmit({
        application_id: applicationId,
        location: values.location,
        deadline: values.deadline.toISOString(),
        slots,
      })
    })
  }

  return (
    <Modal
      title="发起交接预约"
      open={open}
      onCancel={onClose}
      onOk={handleOk}
      confirmLoading={submitting}
      destroyOnClose
      afterOpenChange={(o) => {
        if (o) {
          form.setFieldsValue({
            location: '',
            deadline: dayjs().add(2, 'day').hour(18).minute(0).second(0),
            slots: [{ date: dayjs().add(3, 'day'), range: [dayjs().hour(10).minute(0), dayjs().hour(11).minute(0)] }],
          })
        }
      }}
    >
      <Form form={form} layout="vertical">
        <Form.Item name="location" label="交接地点" rules={[{ required: true, message: '请输入交接地点' }]}>
          <Input placeholder="如：上海市徐汇区xx路xx号 救助站前台" />
        </Form.Item>
        <Form.Item name="deadline" label="确认截止时间" rules={[{ required: true, message: '请选择截止时间' }]}>
          <DatePicker showTime={{ format: 'HH:mm' }} format="YYYY-MM-DD HH:mm" style={{ width: '100%' }} />
        </Form.Item>
        <Form.List name="slots">
          {(fields, { add, remove }) => (
            <>
              {fields.map((field) => (
                <Space key={field.key} align="baseline" style={{ display: 'flex' }}>
                  <Form.Item
                    name={[field.name, 'date']}
                    rules={[{ required: true, message: '日期' }]}
                    style={{ marginBottom: 8 }}
                  >
                    <DatePicker format="YYYY-MM-DD" placeholder="日期" />
                  </Form.Item>
                  <Form.Item
                    name={[field.name, 'range']}
                    rules={[{ required: true, message: '时段' }]}
                    style={{ marginBottom: 8 }}
                  >
                    <TimeRangePicker format="HH:mm" minuteStep={15} />
                  </Form.Item>
                  <DeleteOutlined onClick={() => remove(field.name)} style={{ color: '#ff4d4f' }} />
                </Space>
              ))}
              <Button type="dashed" icon={<PlusOutlined />} onClick={() => add({ date: dayjs().add(3, 'day') })} block>
                添加可预约时段
              </Button>
            </>
          )}
        </Form.List>
      </Form>
    </Modal>
  )
}
