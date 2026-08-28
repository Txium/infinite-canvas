"use client";

import { Alert, Card, Col, Row, Space, Statistic, Typography } from "antd";
import { useEffect, useState } from "react";

import { formatCNY } from "@/constant/credits";
import { fetchAdminFinanceSummary, type AdminFinanceSummary } from "@/services/api/admin";
import { useUserStore } from "@/stores/use-user-store";

export default function AdminFinancePage() {
    const token = useUserStore((state) => state.token);
    const [summary, setSummary] = useState<AdminFinanceSummary | null>(null);
    useEffect(() => { if (token) void fetchAdminFinanceSummary(token).then(setSummary); }, [token]);
    const today = summary?.today;
    const allTime = summary?.allTime;
    return <main className="p-6"><Space direction="vertical" size={16} className="w-full">
        <Alert type="info" showIcon message="当前只展示平台真实钱包流水" description="上游尚未回传实际成本，因此暂不展示利润；供应商余额也不会使用估算数字代替。" />
        <Typography.Title level={4} className="!mb-0">今日</Typography.Title>
        <Row gutter={[16,16]}>
            <Col xs={24} md={12} xl={6}><Card><Statistic title="充值到账" value={formatCNY(today?.rechargeCents || 0)} /></Card></Col>
            <Col xs={24} md={12} xl={6}><Card><Statistic title="生成结算收入" value={formatCNY(today?.revenueCents || 0)} /></Card></Col>
            <Col xs={24} md={12} xl={6}><Card><Statistic title="失败解冻" value={formatCNY(today?.releasedCents || 0)} /></Card></Col>
            <Col xs={24} md={12} xl={6}><Card><Statistic title="成功 / 失败任务" value={`${today?.settledTasks || 0} / ${today?.releasedTasks || 0}`} /></Card></Col>
        </Row>
        <Typography.Title level={4} className="!mb-0">平台总览</Typography.Title>
        <Row gutter={[16,16]}>
            <Col xs={24} md={12} xl={6}><Card><Statistic title="用户数" value={summary?.userCount || 0} /></Card></Col>
            <Col xs={24} md={12} xl={6}><Card><Statistic title="用户可用余额" value={formatCNY(summary?.availableBalanceCents || 0)} /></Card></Col>
            <Col xs={24} md={12} xl={6}><Card><Statistic title="任务冻结余额" value={formatCNY(summary?.frozenBalanceCents || 0)} /></Card></Col>
            <Col xs={24} md={12} xl={6}><Card><Statistic title="累计充值到账" value={formatCNY(allTime?.rechargeCents || 0)} /></Card></Col>
            <Col xs={24} md={12} xl={6}><Card><Statistic title="累计生成收入" value={formatCNY(allTime?.revenueCents || 0)} /></Card></Col>
            <Col xs={24} md={12} xl={6}><Card><Statistic title="累计失败解冻" value={formatCNY(allTime?.releasedCents || 0)} /></Card></Col>
        </Row>
    </Space></main>;
}
