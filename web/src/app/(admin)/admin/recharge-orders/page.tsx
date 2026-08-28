"use client";

import { App, Button, Card, Input, Space, Table, Tag } from "antd";
import { useEffect, useState } from "react";

import { fetchAdminRechargeOrders, reviewAdminRechargeOrder, type RechargeOrder } from "@/services/api/wallet";
import { useUserStore } from "@/stores/use-user-store";

export default function AdminRechargeOrdersPage() {
    const { message, modal } = App.useApp();
    const token = useUserStore((state) => state.token);
    const [items, setItems] = useState<RechargeOrder[]>([]);
    const [keyword, setKeyword] = useState("");
    const load = async () => token && setItems((await fetchAdminRechargeOrders(token, { keyword, page: 1, pageSize: 100 })).items);
    useEffect(() => { void load(); }, [token]);
    const review = (item: RechargeOrder, status: "approved" | "rejected") => modal.confirm({
        title: status === "approved" ? "确认充值到账" : "拒绝充值申请",
        content: status === "approved" ? `将为 ${item.username || item.userId} 增加 ¥${(item.credits / 100).toFixed(2)} 可用余额，此操作不可重复。` : "订单将标记为已拒绝。",
        onOk: async () => { await reviewAdminRechargeOrder(token!, item.id, status); message.success("审核完成"); await load(); },
    });
    return <div className="p-6"><Card title="充值订单" extra={<Space><Input.Search allowClear placeholder="用户/订单/状态" onSearch={(value) => { setKeyword(value); setTimeout(() => void load(), 0); }} /><Button onClick={() => void load()}>刷新</Button></Space>}>
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
    </Card></div>;
}
