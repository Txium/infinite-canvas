"use client";

import { Alert, Card, Col, Row, Space, Statistic, Typography } from "antd";
import { useEffect, useState } from "react";

import { formatCNY } from "@/constant/credits";
import { fetchAdminFinanceSummary, fetchAdminRuntimeReadiness, type AdminFinanceSummary, type AdminRuntimeReadiness } from "@/services/api/admin";
import { useUserStore } from "@/stores/use-user-store";

export default function AdminFinancePage() {
    const token = useUserStore((state) => state.token);
    const [summary, setSummary] = useState<AdminFinanceSummary | null>(null);
	const [readiness, setReadiness] = useState<AdminRuntimeReadiness | null>(null);
    useEffect(() => { if (token) void Promise.all([fetchAdminFinanceSummary(token), fetchAdminRuntimeReadiness(token)]).then(([finance, runtime]) => { setSummary(finance); setReadiness(runtime); }); }, [token]);
    const today = summary?.today;
    const allTime = summary?.allTime;
    return <main className="p-6"><Space direction="vertical" size={16} className="w-full">
		{readiness && !readiness.ready ? <Alert type="warning" showIcon message="平台尚未达到正式收款条件" description={<ul className="mb-0 pl-5">{readiness.issues.map((issue) => <li key={issue}>{issue}</li>)}</ul>} /> : null}
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
		<Typography.Title level={4} className="!mb-0">上线检查</Typography.Title>
		<Row gutter={[16,16]}>
			<Col xs={24} md={8}><Card><Statistic title="数据库" value={readiness?.databasePersistent ? "持久化" : "临时"} valueStyle={{ color: readiness?.databasePersistent ? "#3f8600" : "#cf1322" }} /><Typography.Text type="secondary">驱动：{readiness?.databaseDriver || "检查中"}</Typography.Text></Card></Col>
			<Col xs={24} md={8}><Card><Statistic title="微信 / 支付宝" value={readiness?.paymentConfigured ? "已配置" : "未配置"} valueStyle={{ color: readiness?.paymentConfigured ? "#3f8600" : "#cf1322" }} /></Card></Col>
			<Col xs={24} md={8}><Card><Statistic title="平台托管模型" value={readiness?.managedPlatform ? "已开启" : "未开启"} valueStyle={{ color: readiness?.managedPlatform ? "#3f8600" : "#cf1322" }} /></Card></Col>
		</Row>
    </Space></main>;
}
