"use client";

import { App, Button, Card, Form, InputNumber, Segmented, Statistic, Table, Tag, Typography } from "antd";
import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";

import { createRechargeOrder, fetchRechargeOrders, type RechargeOrder } from "@/services/api/wallet";
import { formatCNY } from "@/constant/credits";
import { useUserStore } from "@/stores/use-user-store";

const statusText = { pending: "待支付", approved: "已到账", rejected: "已关闭" } as const;

export default function WalletPage() {
    const { message } = App.useApp();
    const router = useRouter();
    const token = useUserStore((state) => state.token);
    const user = useUserStore((state) => state.user);
    const isReady = useUserStore((state) => state.isReady);
    const [items, setItems] = useState<RechargeOrder[]>([]);
    const [loading, setLoading] = useState(false);
    const [form] = Form.useForm();
    const load = async () => token && setItems((await fetchRechargeOrders(token)).items);

    useEffect(() => { if (!isReady) return; if (!token) router.replace("/login?redirect=/wallet"); else void load(); }, [isReady, token]);

    const submit = async () => {
        const values = await form.validateFields();
        setLoading(true);
        try {
            const payment = await createRechargeOrder(token!, { amountCents: Math.round(values.amount * 100), paymentMethod: values.paymentMethod, paymentNote: "" });
            message.success("正在打开支付页面，付款成功后自动到账");
            window.location.href = payment.payUrl;
        } finally { setLoading(false); }
    };

    return <main className="h-full min-h-0 overflow-y-auto"><div className="mx-auto max-w-6xl px-4 py-8">
        <Typography.Title level={2}>我的钱包</Typography.Title>
        <div className="grid gap-4 md:grid-cols-3">
            <Card><Statistic title="可用余额" prefix="¥" value={(user?.credits || 0) / 100} precision={2} /><Typography.Text type="secondary">充值多少到账多少，生成时按模型标注售价结算。</Typography.Text></Card>
            <Card><Statistic title="冻结余额" prefix="¥" value={(user?.frozenCredits || 0) / 100} precision={2} /><Typography.Text type="secondary">任务生成中暂时冻结；成功后结算，失败自动退回可用余额。</Typography.Text></Card>
            <Card title="在线充值">
                <Form form={form} layout="vertical" initialValues={{ amount: 10, paymentMethod: "wxpay" }}>
                    <Form.Item name="amount" label="充值金额（元）" rules={[{ required: true }]}><InputNumber min={1} max={100000} className="!w-full" /></Form.Item>
                    <Form.Item name="paymentMethod" label="付款方式"><Segmented options={[{ label: "微信", value: "wxpay" }, { label: "支付宝", value: "alipay" }]} /></Form.Item>
                    <Button type="primary" loading={loading} onClick={() => void submit()}>立即支付</Button>
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
    </div></main>;
}
