"use client";

import { Alert, App, Button, Card, Form, InputNumber, Space, Statistic, Table, Tag, Typography } from "antd";
import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";

import { createRechargeOrder, fetchRechargeOrders, fetchWalletCreditLogs, type CreditLog, type RechargeOrder } from "@/services/api/wallet";
import { apiGet } from "@/services/api/request";
import type { AdminPublicSettings } from "@/services/api/admin";
import { formatCNY } from "@/constant/credits";
import { useUserStore } from "@/stores/use-user-store";
import { useConfigStore } from "@/stores/use-config-store";

const statusText = { pending: "待支付", approved: "已到账", rejected: "已关闭" } as const;
const logText: Record<string, string> = { recharge: "充值到账", ai_freeze: "生成冻结", ai_settle: "生成消费", ai_release: "失败退回", admin_adjust: "余额调整", ai_consume: "生成消费", ai_refund: "生成退款" };

export default function WalletPage() {
    const { message } = App.useApp();
    const router = useRouter();
    const token = useUserStore((state) => state.token);
    const user = useUserStore((state) => state.user);
    const isReady = useUserStore((state) => state.isReady);
    const [items, setItems] = useState<RechargeOrder[]>([]);
    const [logs, setLogs] = useState<CreditLog[]>([]);
	const [loading, setLoading] = useState(false);
	const [paymentState, setPaymentState] = useState<{ ready: boolean; message: string } | null>(null);
    const [form] = Form.useForm();
    const payment = useConfigStore((state) => state.publicSettings?.payment);
    const hydrateUser = useUserStore((state) => state.hydrateUser);
    const load = async () => {
        if (!token) return;
        const [orders, creditLogs] = await Promise.all([
            fetchRechargeOrders(token),
            fetchWalletCreditLogs(token),
        ]);
        setItems(orders.items);
        setLogs(creditLogs.items);
        try {
            const settings = await apiGet<AdminPublicSettings>("/api/settings");
            setPaymentState(settings.payment);
        } catch {
            setPaymentState({ ready: false, message: "支付配置读取失败，请刷新后重试" });
        }
    };

    useEffect(() => {
        if (!isReady) return;
        if (!token) { router.replace("/login?redirect=/wallet"); return; }
        const returned = new URLSearchParams(window.location.search).get("payment") === "return";
        void Promise.all([load(), hydrateUser()]).then(() => {
            if (returned) message.success("支付结果已刷新；若刚完成付款，到账可能需要几秒");
        }).catch((error) => message.error(error instanceof Error ? error.message : "钱包加载失败"));
        if (!returned) return;
        const timer = window.setInterval(() => { void Promise.all([load(), hydrateUser()]); }, 3000);
        const stop = window.setTimeout(() => window.clearInterval(timer), 15000);
        window.history.replaceState(null, "", "/wallet");
        return () => { window.clearInterval(timer); window.clearTimeout(stop); };
    }, [isReady, token]);

    const submit = async () => {
        const values = await form.validateFields();
        setLoading(true);
        try {
            const payment = await createRechargeOrder(token!, { amountCents: Math.round(values.amount * 100), paymentMethod: values.paymentMethod, paymentNote: "" });
            message.success("正在打开支付页面，付款成功后自动到账");
            window.location.href = payment.payUrl;
        } catch (error) {
            message.error(error instanceof Error ? error.message : "创建支付订单失败");
        } finally { setLoading(false); }
    };

    return <main className="h-full min-h-0 overflow-y-auto"><div className="mx-auto max-w-6xl px-4 py-8">
        <Typography.Title level={2}>我的钱包</Typography.Title>
        <div className="grid gap-4 md:grid-cols-3">
            <Card><Statistic title="可用余额" prefix="¥" value={(user?.credits || 0) / 100} precision={2} /><Typography.Text type="secondary">充值多少到账多少，生成时按模型标注售价结算。</Typography.Text></Card>
            <Card><Statistic title="冻结余额" prefix="¥" value={(user?.frozenCredits || 0) / 100} precision={2} /><Typography.Text type="secondary">任务生成中暂时冻结；成功后结算，失败自动退回可用余额。</Typography.Text></Card>
            <Card title="在线充值">
				{(paymentState || payment) && !(paymentState || payment)?.ready ? <Alert type="warning" showIcon message="在线充值暂未开放" description={(paymentState || payment)?.message || "请稍后重试"} className="mb-4" /> : null}
                <Form form={form} layout="vertical" initialValues={{ amount: 10, paymentMethod: "alipay" }}>
                    <Space wrap className="mb-3">{[10, 20, 50, 100].map((amount) => <Button key={amount} onClick={() => form.setFieldValue("amount", amount)}>¥{amount}</Button>)}</Space>
                    <Form.Item name="amount" label="充值金额（元）" rules={[{ required: true }]}><InputNumber min={1} max={100000} className="!w-full" /></Form.Item>
                    <Form.Item name="paymentMethod" hidden><input type="hidden" /></Form.Item>
                    <Typography.Paragraph>付款方式：支付宝</Typography.Paragraph>
                    <Alert
                        type="info"
                        showIcon
                        className="mb-3"
                        message="测试支付请使用其他支付宝账号"
                        description="不要用签约商户账号给自己付款，否则支付宝可能返回 AE150003030。"
                    />
                    <Button type="primary" disabled={(paymentState || payment)?.ready !== true} loading={loading} onClick={() => void submit()}>立即支付</Button>
                </Form>
            </Card>
        </div>
        <Card title="充值记录" className="mt-5">
            <Table rowKey="id" dataSource={items} pagination={false} columns={[
                { title: "时间", dataIndex: "createdAt" },
                { title: "金额", dataIndex: "amountCents", render: (value: number) => `¥${(value / 100).toFixed(2)}` },
                { title: "到账余额", dataIndex: "credits", render: (value: number) => formatCNY(value) },
                { title: "状态", dataIndex: "status", render: (value: RechargeOrder["status"]) => <Tag color={value === "approved" ? "success" : value === "rejected" ? "error" : "processing"}>{statusText[value]}</Tag> },
                { title: "备注", dataIndex: "paymentNote" },
                { title: "管理员备注", dataIndex: "adminRemark" },
            ]} />
        </Card>
        <Card title="钱包流水" className="mt-5">
            <Table rowKey="id" dataSource={logs} pagination={{ pageSize: 10 }} columns={[
                { title: "时间", dataIndex: "createdAt" },
                { title: "类型", dataIndex: "type", render: (value: string) => logText[value] || value },
                { title: "可用余额变化", dataIndex: "amount", render: (value: number) => <Typography.Text type={value < 0 ? "danger" : value > 0 ? "success" : undefined}>{value > 0 ? "+" : ""}{formatCNY(value)}</Typography.Text> },
                { title: "冻结变化", dataIndex: "frozenAmount", render: (value: number) => `${value > 0 ? "+" : ""}${formatCNY(value)}` },
                { title: "可用余额", dataIndex: "balance", render: (value: number) => formatCNY(value) },
                { title: "说明", dataIndex: "remark" },
            ]} />
        </Card>
    </div></main>;
}
