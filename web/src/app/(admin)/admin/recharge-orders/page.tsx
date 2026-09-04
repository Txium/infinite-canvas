"use client";

import { App, Button, Card, Input, Space, Table, Tag } from "antd";
import { useEffect, useState } from "react";

import { fetchAdminRechargeOrders, fetchAdminRefundOrders, reviewAdminRechargeOrder, reviewAdminRefundOrder, type RechargeOrder, type RefundOrder } from "@/services/api/wallet";
import { useUserStore } from "@/stores/use-user-store";

export default function AdminRechargeOrdersPage() {
    const { message, modal } = App.useApp();
    const token = useUserStore((state) => state.token);
    const [items, setItems] = useState<RechargeOrder[]>([]);
	const [refunds, setRefunds] = useState<RefundOrder[]>([]);
    const [keyword, setKeyword] = useState("");
	const load = async () => {
		if (!token) return;
		const [orders, refundOrders] = await Promise.all([fetchAdminRechargeOrders(token, { keyword, page: 1, pageSize: 100 }), fetchAdminRefundOrders(token, { keyword, page: 1, pageSize: 100 })]);
		setItems(orders.items);
		setRefunds(refundOrders.items);
	};
    useEffect(() => { void load(); }, [token]);
    const review = (item: RechargeOrder, status: "approved" | "rejected") => modal.confirm({
        title: status === "approved" ? "确认充值到账" : "拒绝充值申请",
        content: status === "approved" ? `将为 ${item.username || item.userId} 增加 ¥${(item.credits / 100).toFixed(2)} 可用余额，此操作不可重复。` : "订单将标记为已拒绝。",
        onOk: async () => { await reviewAdminRechargeOrder(token!, item.id, status); message.success("审核完成"); await load(); },
    });
	const reviewRefund = (item: RefundOrder, action: "approve" | "reject" | "query") => modal.confirm({
		title: action === "approve" ? "确认支付宝原路退款" : action === "reject" ? "拒绝退款申请" : "查询支付宝退款结果",
		content: action === "approve" ? `将向支付宝提交 ¥${(item.amountCents / 100).toFixed(2)} 原路退款。确认前请核对订单与买家申请，此操作可能真实退回资金。` : action === "reject" ? "拒绝后，已锁定金额会自动恢复至买家可用余额。" : "将使用原退款请求号向支付宝查询，避免重复退款。",
		onOk: async () => { try { await reviewAdminRefundOrder(token!, item.id, action); message.success(action === "query" ? "退款结果已同步" : "退款审核已处理"); } finally { await load(); } },
	});
    return <div className="space-y-5 p-6"><Card title="充值订单" extra={<Space><Input.Search allowClear placeholder="用户/订单/状态" onSearch={(value) => { setKeyword(value); setTimeout(() => void load(), 0); }} /><Button onClick={() => void load()}>刷新</Button></Space>}>
        <Table rowKey="id" dataSource={items} columns={[
            { title: "用户", render: (_: unknown, item: RechargeOrder) => item.username || item.userId },
            { title: "金额", dataIndex: "amountCents", render: (value: number) => `¥${(value / 100).toFixed(2)}` },
            { title: "到账余额", dataIndex: "credits", render: (value: number) => `¥${(value / 100).toFixed(2)}` },
            { title: "方式", dataIndex: "paymentMethod" },
            { title: "付款备注", dataIndex: "paymentNote" },
            { title: "状态", dataIndex: "status", render: (value: string) => <Tag color={value === "approved" ? "success" : value === "rejected" ? "error" : "processing"}>{value}</Tag> },
            { title: "时间", dataIndex: "createdAt" },
            { title: "操作", render: (_: unknown, item: RechargeOrder) => item.status === "pending" ? <Space><Button type="primary" size="small" onClick={() => review(item, "approved")}>确认到账</Button><Button danger size="small" onClick={() => review(item, "rejected")}>拒绝</Button></Space> : null },
        ]} />
	</Card>
	<Card title="退款申请">
		<Table rowKey="id" dataSource={refunds} columns={[
			{ title: "用户", render: (_: unknown, item: RefundOrder) => item.username || item.userId },
			{ title: "退款金额", dataIndex: "amountCents", render: (value: number) => `¥${(value / 100).toFixed(2)}` },
			{ title: "原因", dataIndex: "reason" },
			{ title: "状态", dataIndex: "status", render: (value: RefundOrder["status"]) => <Tag color={value === "succeeded" ? "success" : value === "rejected" || value === "failed" ? "error" : "processing"}>{value}</Tag> },
			{ title: "说明", render: (_: unknown, item: RefundOrder) => item.failureMessage || item.adminRemark || "-" },
			{ title: "时间", dataIndex: "createdAt" },
			{ title: "操作", render: (_: unknown, item: RefundOrder) => item.status === "pending" ? <Space><Button danger type="primary" size="small" onClick={() => reviewRefund(item, "approve")}>原路退款</Button><Button size="small" onClick={() => reviewRefund(item, "reject")}>拒绝</Button></Space> : item.status === "processing" ? <Space><Button size="small" onClick={() => reviewRefund(item, "query")}>查询支付宝结果</Button><Button danger size="small" onClick={() => reviewRefund(item, "approve")}>使用原请求号重试</Button></Space> : null },
		]} />
	</Card></div>;
}
