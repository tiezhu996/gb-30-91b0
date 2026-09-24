import { Button, Card, Col, DatePicker, Empty, Form, Input, Popconfirm, Row, Select, Space, Tag, message } from 'antd'
import dayjs from 'dayjs'
import { useEffect, useState } from 'react'
import {
  cancelHandover,
  confirmHandover,
  listApplicationHandovers,
  offerHandover,
  selectHandoverSlot,
} from '@/api/handover'
import { HandoverCancelledByText, HandoverStatusMap } from '@/constants/handover'
import type { HandoverAppointment } from '@/types/api'
import { formatDateTime } from '@/utils/dateFormat'

/**
 * HandoverPanel renders the full handover appointment history of one
 * application and the actions available to the adopter / org.
 */
export default function HandoverPanel({
  applicationId,
  approved,
  isOrg,
}: {
  applicationId: number
  approved: boolean
  isOrg: boolean
}) {
  const [items, setItems] = useState<HandoverAppointment[]>([])
  const [offering, setOffering] = useState(false)
  const [form] = Form.useForm()
  const [loading, setLoading] = useState(false)

  async function load() {
    setItems(await listApplicationHandovers(applicationId))
  }
  useEffect(() => {
    load().catch(() => undefined)
  }, [applicationId])

  const active = items.find((h) => h.status === 'offered' || h.status === 'pending')

  // Hourly candidate slots between the picked begin/end, defaulting to the
  // upcoming 7 days.
  const slotOptions = (() => {
    const range: [dayjs.Dayjs, dayjs.Dayjs] = form.getFieldValue('slotRange')
    if (!range) return []
    const [start, end] = range
    const out: { value: string; label: string }[] = []
    for (let cur = start.clone(); cur.isBefore(end); cur = cur.add(1, 'hour')) {
      out.push({ value: cur.toISOString(), label: cur.format('MM-DD HH:mm') })
    }
    return out
  })()

  async function onOffer() {
    const values = await form.validateFields()
    setLoading(true)
    try {
      await offerHandover(applicationId, {
        location: values.location,
        slots: values.slots as string[],
        confirm_before: values.deadline.toISOString(),
      })
      message.success('交接预约已发起')
      setOffering(false)
      form.resetFields()
      await load()
    } finally {
      setLoading(false)
    }
  }

  async function onSelect(h: HandoverAppointment, slot: string) {
    await selectHandoverSlot(h.id, slot)
    message.success('时段已占用，等待机构确认')
    await load()
  }

  async function onConfirm(h: HandoverAppointment) {
    await confirmHandover(h.id)
    message.success('预约已确认')
    await load()
  }

  async function onCancel(h: HandoverAppointment) {
    await cancelHandover(h.id)
    message.success('预约已取消，时段已释放')
    await load()
  }

  return (
    <div style={{ marginTop: 8 }}>
      <Space style={{ marginBottom: 8 }}>
        <strong>交接预约</strong>
        {isOrg && approved && !active && (
          <Button size="small" type="primary" onClick={() => setOffering((v) => !v)}>
            {offering ? '收起' : items.length ? '重新安排' : '发起交接预约'}
          </Button>
        )}
      </Space>

      {offering && (
        <Card size="small" style={{ marginBottom: 12 }}>
          <Form form={form} layout="vertical">
            <Form.Item name="location" label="交接地点" rules={[{ required: true, message: '请输入交接地点' }]}>
              <Input placeholder="例如：上海市徐汇区救助站领养大厅" />
            </Form.Item>
            <Row gutter={8}>
              <Col span={12}>
                <Form.Item
                  name="deadline"
                  label="确认截止时间"
                  rules={[{ required: true, message: '请选择截止时间' }]}
                >
                  <DatePicker showTime style={{ width: '100%' }} />
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item
                  name="slotRange"
                  label="时段范围（用于生成候选）"
                  rules={[{ required: true, message: '请选择时段范围' }]}
                >
                  <DatePicker.RangePicker
                    showTime={{ format: 'HH:mm' }}
                    style={{ width: '100%' }}
                    onChange={() => form.setFieldValue('slots', [])}
                  />
                </Form.Item>
              </Col>
            </Row>
            <Form.Item
              noStyle
              shouldUpdate={(prev, cur) => prev.slotRange !== cur.slotRange}
            >
              {() => (
                <Form.Item
                  name="slots"
                  label="可预约时段（可多选，按整点）"
                  rules={[{ required: true, message: '请至少选择一个时段' }]}
                >
                  <Select mode="multiple" options={slotOptions} placeholder="先选择时段范围，再勾选整点时段" />
                </Form.Item>
              )}
            </Form.Item>
            <Space>
              <Button type="primary" loading={loading} onClick={onOffer}>
                发起预约
              </Button>
              <Button
                onClick={() => {
                  setOffering(false)
                  form.resetFields()
                }}
              >
                取消
              </Button>
            </Space>
          </Form>
        </Card>
      )}

      {items.length === 0 && <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无交接预约" />}

      {items.map((h) => {
        const meta = HandoverStatusMap[h.status]
        return (
          <Card key={h.id} size="small" style={{ marginBottom: 8 }}>
            <Space wrap style={{ marginBottom: 6 }}>
              <Tag color={meta.color}>{meta.text}</Tag>
              <span>地点：{h.location}</span>
              <span>确认截止：{formatDateTime(h.confirm_before)}</span>
              {h.selected_slot && <span>已选时段：{formatDateTime(h.selected_slot)}</span>}
              {h.status === 'cancelled' && h.cancelled_by && (
                <span>取消方：{HandoverCancelledByText[h.cancelled_by] || h.cancelled_by}</span>
              )}
              {h.confirmed_at && <span>确认时间：{formatDateTime(h.confirmed_at)}</span>}
            </Space>

            {h.status === 'offered' && (
              <Select
                size="small"
                style={{ minWidth: 260 }}
                placeholder="选择可预约时段"
                disabled={isOrg}
                options={h.slots.map((s) => ({
                  value: s.time,
                  disabled: s.locked,
                  label: `${formatDateTime(s.time)}${s.locked ? '（已占用）' : ''}`,
                }))}
                onChange={(v) => onSelect(h, v)}
              />
            )}

            <Space style={{ marginTop: 6 }}>
              {isOrg && h.status === 'pending' && (
                <Popconfirm title="确认该交接预约？" onConfirm={() => onConfirm(h)}>
                  <Button size="small" type="primary">
                    确认预约
                  </Button>
                </Popconfirm>
              )}
              {(h.status === 'offered' || h.status === 'pending') && (
                <Popconfirm title="取消后已选时段将释放，确定取消？" onConfirm={() => onCancel(h)}>
                  <Button size="small" danger>
                    取消预约
                  </Button>
                </Popconfirm>
              )}
            </Space>
          </Card>
        )
      })}
    </div>
  )
}
