"use client";

import { App, Button, Card, Form, Input, InputNumber, Modal, Space, Switch, Table, Tag, Typography } from "antd";
import dayjs from "dayjs";
import { useEffect, useState } from "react";

import { fetchAdminModelProviders, saveAdminModelProvider, type AdminModelProvider } from "@/services/api/admin";
import { useUserStore } from "@/stores/use-user-store";

type ProviderForm = Omit<AdminModelProvider, "balanceCents" | "warningBalanceCents" | "criticalBalanceCents" | "lowBalanceCents"> & {
    balanceYuan?: number;
    warningBalanceYuan: number;
    criticalBalanceYuan: number;
    lowBalanceYuan: number;
};

const statusMeta: Record<AdminModelProvider["balanceStatus"], { color: string; label: string }> = {
    disabled: { color: "default", label: "已停用" },
    not_ready: { color: "error", label: "未配置" },
    unknown: { color: "processing", label: "待查看" },
    very_low: { color: "error", label: "极低余额" },
    critical: { color: "error", label: "余额不足" },
    warning: { color: "warning", label: "余额偏低" },
    normal: { color: "success", label: "余额正常" },
};

const yuan = (value: number | null | undefined) => value == null ? "未记录" : `¥${(value / 100).toFixed(2)}`;

export default function AdminProvidersPage() {
    const { message } = App.useApp();
    const token = useUserStore((state) => state.token);
    const [items, setItems] = useState<AdminModelProvider[]>([]);
    const [editing, setEditing] = useState<AdminModelProvider | null>(null);
    const [form] = Form.useForm<ProviderForm>();
    const load = async () => { if (token) setItems(await fetchAdminModelProviders(token)); };
    useEffect(() => { void load(); }, [token]);
    const open = (item: AdminModelProvider) => {
        setEditing(item);
        form.setFieldsValue({
            ...item,
            balanceYuan: item.balanceCents == null ? undefined : item.balanceCents / 100,
            warningBalanceYuan: item.warningBalanceCents / 100,
            criticalBalanceYuan: item.criticalBalanceCents / 100,
            lowBalanceYuan: item.lowBalanceCents / 100,
        });
    };
    const save = async () => {
        if (!editing || !token) return;
        const values = await form.validateFields();
        const { balanceYuan, warningBalanceYuan, criticalBalanceYuan, lowBalanceYuan, ...providerValues } = values;
        await saveAdminModelProvider(token, {
            ...editing,
            ...providerValues,
            balanceCents: balanceYuan == null ? null : Math.round(balanceYuan * 100),
            warningBalanceCents: Math.round(warningBalanceYuan * 100),
            criticalBalanceCents: Math.round(criticalBalanceYuan * 100),
            lowBalanceCents: Math.round(lowBalanceYuan * 100),
        });
        message.success("中转站配置已保存");
        setEditing(null);
        await load();
    };
    return <div className="p-6">
        <Card title="四家上游中转站" extra={<Button onClick={() => void load()}>刷新</Button>}>
            <Typography.Paragraph type="secondary">API Key 只保存在 Render Secret。余额为管理员从上游核实后的手工记录；未记录时明确提示登录上游查看，不使用估算值。</Typography.Paragraph>
            <Table rowKey="id" dataSource={items} pagination={false} columns={[
                { title: "中转站", render: (_: unknown, item: AdminModelProvider) => <Space direction="vertical" size={0}><Typography.Text strong>{item.name}</Typography.Text><Typography.Text type="secondary">{item.code}</Typography.Text></Space> },
                { title: "连接配置", render: (_: unknown, item: AdminModelProvider) => <Space direction="vertical" size={2}><Tag color={item.hasApiKey ? "success" : "error"}>{item.hasApiKey ? "服务器 Key 已配置" : "服务器 Key 未配置"}</Tag><Typography.Text type="secondary" ellipsis style={{ maxWidth: 260 }}>{item.baseUrl || "Base URL 未配置"}</Typography.Text></Space> },
                { title: "线路", render: (_: unknown, item: AdminModelProvider) => `${item.enabledRouteCount} 启用 / ${item.routeCount} 总计` },
                { title: "记录余额", render: (_: unknown, item: AdminModelProvider) => <Space direction="vertical" size={0}><Typography.Text strong>{yuan(item.balanceCents)}</Typography.Text><Typography.Text type="secondary">{item.balanceCheckedAt ? dayjs(item.balanceCheckedAt).format("YYYY-MM-DD HH:mm") : "需要登录上游查看"}</Typography.Text></Space> },
                { title: "预警", render: (_: unknown, item: AdminModelProvider) => { const meta = statusMeta[item.balanceStatus] || statusMeta.unknown; return <Space direction="vertical" size={0}><Tag color={meta.color}>{meta.label}</Tag><Typography.Text type="secondary">{item.balanceMessage || "需要登录上游查看"}</Typography.Text></Space>; } },
                { title: "状态", render: (_: unknown, item: AdminModelProvider) => <Tag color={item.ready ? "success" : "default"}>{item.ready ? "可路由" : "不可路由"}</Tag> },
                { title: "操作", render: (_: unknown, item: AdminModelProvider) => <Button size="small" onClick={() => open(item)}>配置 / 更新余额</Button> },
            ]} />
        </Card>
        <Modal title={editing ? `配置 ${editing.name}` : "配置中转站"} open={!!editing} onCancel={() => setEditing(null)} onOk={() => void save()} width={620}>
            <Form form={form} layout="vertical">
                <Form.Item name="baseUrl" label="Base URL"><Input placeholder="填写已核实的上游 API Base URL" /></Form.Item>
                <Form.Item label="API Key"><div className="rounded-lg border border-stone-200 p-3 text-sm text-stone-500 dark:border-stone-700">只在 Render 环境变量中配置对应的 <code>MODEL_PROVIDER_*_API_KEY</code>，本页面不读取明文。</div></Form.Item>
                <Form.Item name="balanceYuan" label="当前上游余额（人民币，手工核实）"><InputNumber min={0} precision={2} className="w-full" placeholder="留空表示需要登录上游查看" /></Form.Item>
                <Space align="start" className="w-full" size={12}>
                    <Form.Item name="warningBalanceYuan" label="黄色提醒（元）" rules={[{ required: true }]}><InputNumber min={0.01} precision={2} /></Form.Item>
                    <Form.Item name="criticalBalanceYuan" label="红色提醒（元）" rules={[{ required: true }]}><InputNumber min={0.01} precision={2} /></Form.Item>
                    <Form.Item name="lowBalanceYuan" label="极低余额（元）" rules={[{ required: true }]}><InputNumber min={0.01} precision={2} /></Form.Item>
                </Space>
                <Space align="start" className="w-full" size={12}>
                    <Form.Item name="priority" label="排序"><InputNumber min={1} precision={0} /></Form.Item>
                    <Form.Item name="timeout" label="超时秒数"><InputNumber min={30} precision={0} /></Form.Item>
                    <Form.Item name="enabled" label="启用供应商" valuePropName="checked"><Switch /></Form.Item>
                </Space>
            </Form>
        </Modal>
    </div>;
}
