"use client";

import { App, Button, Card, Form, Input, InputNumber, Modal, Space, Switch, Table, Tag, Typography } from "antd";
import dayjs from "dayjs";
import { useEffect, useRef, useState } from "react";

import { fetchAdminModelProviders, fetchAdminProviderLedgers, recordAdminProviderTopup, saveAdminModelProvider, syncAdminModelProviderCatalog, testAdminModelProvider, type AdminModelProvider, type AdminProviderLedger } from "@/services/api/admin";
import { useUserStore } from "@/stores/use-user-store";

type ProviderForm = Omit<AdminModelProvider, "balanceCents" | "warningBalanceCents" | "criticalBalanceCents" | "lowBalanceCents"> & {
    warningBalanceYuan: number;
    criticalBalanceYuan: number;
    lowBalanceYuan: number;
};
type TopupForm = { amountYuan: number; reason: string; reference?: string };

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
    const [ledgers, setLedgers] = useState<AdminProviderLedger[]>([]);
    const [editing, setEditing] = useState<AdminModelProvider | null>(null);
    const [testingID, setTestingID] = useState("");
    const [syncingID, setSyncingID] = useState("");
    const [balanceErrors, setBalanceErrors] = useState<Record<string, string>>({});
    const autoBalanceKey = useRef("");
	const balancePending = useRef(new Set<string>());
	const loadSequence = useRef(0);
	const [saving, setSaving] = useState(false);
	const [topupSaving, setTopupSaving] = useState(false);
	const savingRef = useRef(false);
	const topupSavingRef = useRef(false);
    const [testedModels, setTestedModels] = useState<{ provider: string; models: string[] } | null>(null);
    const [form] = Form.useForm<ProviderForm>();
    const [topupForm] = Form.useForm<TopupForm>();
    const [topupProvider, setTopupProvider] = useState<AdminModelProvider | null>(null);
    const load = async () => {
		const sequence = ++loadSequence.current;
		if (!token) return;
		try {
			const [providers, history] = await Promise.all([fetchAdminModelProviders(token), fetchAdminProviderLedgers(token)]);
			if (sequence !== loadSequence.current || useUserStore.getState().token !== token) return;
			setItems(providers || []); setLedgers(history || []);
		} catch { if (useUserStore.getState().token === token) void message.error("中转站数据读取失败，请刷新重试"); }
	};
    useEffect(() => { setItems([]); setLedgers([]); setBalanceErrors({}); autoBalanceKey.current = ""; balancePending.current = new Set(); void load(); return () => { loadSequence.current++; }; }, [token]);
    useEffect(() => {
        if (!token || items.length === 0) return;
        const key = token + items.map((item) => `${item.id}:${item.ready}:${item.baseUrl}`).join("|");
        if (autoBalanceKey.current === key) return;
        autoBalanceKey.current = key;
        void Promise.all(items.filter((item) => item.ready).map((item) => testConnection(item, false)));
    }, [items, token]);
    const open = (item: AdminModelProvider) => {
        setEditing(item);
        form.setFieldsValue({
            ...item,
            warningBalanceYuan: item.warningBalanceCents / 100,
            criticalBalanceYuan: item.criticalBalanceCents / 100,
            lowBalanceYuan: item.lowBalanceCents / 100,
        });
    };
    const save = async () => {
        if (!editing || !token || savingRef.current) return;
		savingRef.current = true; setSaving(true);
		try {
        const values = await form.validateFields();
        const { warningBalanceYuan, criticalBalanceYuan, lowBalanceYuan, ...providerValues } = values;
        await saveAdminModelProvider(token, {
            ...editing,
            ...providerValues,
            warningBalanceCents: Math.round(warningBalanceYuan * 100),
            criticalBalanceCents: Math.round(criticalBalanceYuan * 100),
            lowBalanceCents: Math.round(lowBalanceYuan * 100),
        });
        message.success("中转站配置已保存");
        setEditing(null);
        await load();
		} catch (error) { if (!(error && typeof error === "object" && "errorFields" in error)) void message.error(error instanceof Error ? error.message : "保存失败"); }
		finally { savingRef.current = false; setSaving(false); }
    };
    const saveTopup = async () => {
        if (!topupProvider || !token || topupSavingRef.current) return;
		topupSavingRef.current = true; setTopupSaving(true);
		try {
        const values = await topupForm.validateFields();
        await recordAdminProviderTopup(token, { providerId: topupProvider.id, amountCents: Math.round(values.amountYuan * 100), reason: values.reason, reference: values.reference });
        message.success("上游充值已登记并写入独立资金账本");
        setTopupProvider(null);
        topupForm.resetFields();
        await load();
		} catch (error) { if (!(error && typeof error === "object" && "errorFields" in error)) void message.error(error instanceof Error ? error.message : "登记失败，请先核对流水再重试"); }
		finally { topupSavingRef.current = false; setTopupSaving(false); }
    };
    const testConnection = async (item: AdminModelProvider, notify = true) => {
        if (!token || balancePending.current.has(item.id)) return;
		balancePending.current.add(item.id);
        if (notify) setTestingID(item.id);
        try {
            const result = await testAdminModelProvider(token, item.id);
			if (useUserStore.getState().token !== token) return;
            if (notify) void message.success([result.message, result.balanceText].filter(Boolean).join("；"));
			if (result.balanceAmount != null && result.balanceCurrency && result.balanceCheckedAt) setItems((current) => current.map((provider) => provider.id === item.id ? { ...provider, upstreamBalanceAmount: result.balanceAmount, upstreamBalanceCurrency: result.balanceCurrency, upstreamBalanceCheckedAt: result.balanceCheckedAt, upstreamBalanceAttemptedAt: result.balanceCheckedAt, upstreamBalanceError: "" } : provider));
            setBalanceErrors((current) => { const next = { ...current }; delete next[item.id]; return next; });
            if (notify && result.models?.length) setTestedModels({ provider: item.name, models: result.models });
        } catch (error) {
			if (useUserStore.getState().token !== token) return;
            const detail = error instanceof Error ? error.message : "余额查询失败，请检查上游 Key 权限";
            setBalanceErrors((current) => ({ ...current, [item.id]: detail }));
            if (notify) void message.error(detail);
        } finally {
			if (useUserStore.getState().token === token) { balancePending.current.delete(item.id); if (notify) setTestingID(""); }
        }
    };
    const syncCatalog = async (item: AdminModelProvider) => {
        if (!token) return;
        setSyncingID(item.id);
        try {
            const result = await syncAdminModelProviderCatalog(token, item.id);
            message.success(`已核对 ${result.checkedRoutes} 条线路，停用 ${result.disabledRoutes.length} 条上游已不存在的线路`);
            await load();
        } catch (error) {
            message.error(error instanceof Error ? error.message : "模型目录同步失败，本次未修改线路");
        } finally {
            setSyncingID("");
        }
    };
    return <div className="p-6">
        <Card title="四家上游中转站" extra={<Button onClick={() => void load()}>刷新</Button>}>
            <Typography.Paragraph type="secondary">API Key 只保存在服务器 Secret。查询余额会独立保存上游原币数值和时间，失败时保留最近一次核实值。人民币手工资金账单独显示，预警阈值针对人民币账本；LEC 当前仅测试模型接口，不伪造余额。</Typography.Paragraph>
            <Table rowKey="id" dataSource={items} pagination={false} columns={[
                { title: "中转站", render: (_: unknown, item: AdminModelProvider) => <Space direction="vertical" size={0}><Typography.Text strong>{item.name}</Typography.Text><Typography.Text type="secondary">{item.code}</Typography.Text></Space> },
                { title: "连接配置", render: (_: unknown, item: AdminModelProvider) => <Space direction="vertical" size={2}><Tag color={item.hasApiKey ? "success" : "error"}>{item.hasApiKey ? "服务器 Key 已配置" : "服务器 Key 未配置"}</Tag><Typography.Text type="secondary" ellipsis style={{ maxWidth: 260 }}>{item.baseUrl || "Base URL 未配置"}</Typography.Text></Space> },
                { title: "线路", render: (_: unknown, item: AdminModelProvider) => `${item.enabledRouteCount} 启用 / ${item.routeCount} 总计` },
                { title: "上游核实余额", render: (_: unknown, item: AdminModelProvider) => <Space direction="vertical" size={0}><Typography.Text strong>{item.upstreamBalanceAmount != null && item.upstreamBalanceAmount !== "" && item.upstreamBalanceCurrency ? `${item.upstreamBalanceAmount} ${item.upstreamBalanceCurrency}` : "未核实"}</Typography.Text><Typography.Text type="secondary">{item.upstreamBalanceCheckedAt ? `最近成功：${dayjs(item.upstreamBalanceCheckedAt).format("YYYY-MM-DD HH:mm:ss")}` : item.code === "lec" ? "上游未提供余额接口" : "点击查询余额"}</Typography.Text>{balanceErrors[item.id] || item.upstreamBalanceError ? <Typography.Text type="danger">{balanceErrors[item.id] || item.upstreamBalanceError}</Typography.Text> : null}</Space> },
                { title: "人民币手工账", render: (_: unknown, item: AdminModelProvider) => <Space direction="vertical" size={0}><Typography.Text>{yuan(item.balanceCents)}</Typography.Text><Typography.Text type="secondary">{item.balanceCheckedAt ? dayjs(item.balanceCheckedAt).format("YYYY-MM-DD HH:mm") : "未登记"}</Typography.Text></Space> },
                { title: "预警", render: (_: unknown, item: AdminModelProvider) => { const meta = statusMeta[item.balanceStatus] || statusMeta.unknown; return <Space direction="vertical" size={0}><Tag color={meta.color}>{meta.label}</Tag><Typography.Text type="secondary">{item.balanceMessage || "需要登录上游查看"}</Typography.Text></Space>; } },
                { title: "状态", render: (_: unknown, item: AdminModelProvider) => <Tag color={item.ready ? "success" : "default"}>{item.ready ? "可路由" : "不可路由"}</Tag> },
                { title: "操作", render: (_: unknown, item: AdminModelProvider) => <Space wrap><Button size="small" loading={testingID === item.id} disabled={!item.ready} onClick={() => void testConnection(item)}>查询余额</Button><Button size="small" loading={syncingID === item.id} disabled={!item.ready || !["302", "lec", "wavespeed"].includes(item.code)} onClick={() => void syncCatalog(item)}>同步模型</Button><Button size="small" onClick={() => open(item)}>配置</Button><Button size="small" type="primary" onClick={() => setTopupProvider(item)}>登记充值</Button></Space> },
            ]} />
        </Card>
        <Card title="上游资金流水" className="mt-4">
            <Table rowKey="id" size="small" dataSource={ledgers} pagination={{ pageSize: 10 }} columns={[
                { title: "时间", dataIndex: "createdAt", render: (value: string) => dayjs(value).format("YYYY-MM-DD HH:mm") },
                { title: "Provider", dataIndex: "providerId" },
                { title: "类型", dataIndex: "type" },
                { title: "金额", render: (_: unknown, item: AdminProviderLedger) => `${item.currency} ${(item.amount / 100).toFixed(2)}` },
                { title: "余额前 / 后", render: (_: unknown, item: AdminProviderLedger) => `${(item.balanceBefore / 100).toFixed(2)} / ${(item.balanceAfter / 100).toFixed(2)}` },
                { title: "原因", dataIndex: "reason" },
                { title: "凭证", dataIndex: "reference", render: (value: string) => value || "-" },
            ]} />
        </Card>
        <Modal title={editing ? `配置 ${editing.name}` : "配置中转站"} open={!!editing} confirmLoading={saving} cancelButtonProps={{disabled:saving}} closable={!saving} maskClosable={!saving} keyboard={!saving} onCancel={() => { if (!savingRef.current) setEditing(null); }} onOk={() => void save()} width={620}>
            <Form form={form} layout="vertical">
                <Form.Item name="baseUrl" label="Base URL"><Input placeholder="填写已核实的上游 API Base URL" /></Form.Item>
                <Form.Item label="API Key"><div className="rounded-lg border border-stone-200 p-3 text-sm text-stone-500 dark:border-stone-700">只在 Render 环境变量中配置对应的 <code>MODEL_PROVIDER_*_API_KEY</code>，本页面不读取明文。</div></Form.Item>
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
        <Modal title={topupProvider ? `登记 ${topupProvider.name} 上游充值` : "登记上游充值"} open={!!topupProvider} confirmLoading={topupSaving} cancelButtonProps={{disabled:topupSaving}} closable={!topupSaving} maskClosable={!topupSaving} keyboard={!topupSaving} onCancel={() => { if (!topupSavingRef.current) setTopupProvider(null); }} onOk={() => void saveTopup()}>
            <Form form={topupForm} layout="vertical">
                <Form.Item name="amountYuan" label="实际充值金额（人民币元）" rules={[{ required: true, message: "请输入充值金额" }]}><InputNumber min={0.01} precision={2} className="w-full" /></Form.Item>
                <Form.Item name="reason" label="原因" rules={[{ required: true, whitespace: true, message: "请填写原因" }]}><Input placeholder="例如：LEC 余额低于预警线" /></Form.Item>
                <Form.Item name="reference" label="凭证 / 订单参考"><Input placeholder="可填写上游订单号或凭证链接" /></Form.Item>
                <Typography.Text type="secondary">本操作只登记已由管理员手工完成的上游充值，不会自动调用支付宝或模拟网页付款。</Typography.Text>
            </Form>
        </Modal>
        <Modal title={`${testedModels?.provider || "中转站"} 实际返回模型`} open={!!testedModels} footer={<Button type="primary" onClick={() => setTestedModels(null)}>知道了</Button>} onCancel={() => setTestedModels(null)} width={760}>
            <Typography.Paragraph type="secondary">以下是上游接口当前真实返回的模型 ID，可用于核对线路；这里只显示模型名，不显示 API Key。</Typography.Paragraph>
            <div className="max-h-[55vh] overflow-auto rounded-lg border border-stone-200 p-3 dark:border-stone-700"><Space wrap>{testedModels?.models.map((model) => <Tag key={model}>{model}</Tag>)}</Space></div>
        </Modal>
    </div>;
}
