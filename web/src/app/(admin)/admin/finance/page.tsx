"use client";

import { Alert, App, Button, Card, Col, DatePicker, Form, Input, InputNumber, Modal, Row, Select, Space, Statistic, Table, Typography } from "antd";
import { useEffect, useState } from "react";

import { formatCNY } from "@/constant/credits";
import { fetchAdminFinanceSummary, fetchAdminOperatingExpenses, fetchAdminRuntimeReadiness, recordAdminOperatingExpense, type AdminFinanceSummary, type AdminOperatingExpense, type AdminRuntimeReadiness } from "@/services/api/admin";
import { useUserStore } from "@/stores/use-user-store";

export default function AdminFinancePage() {
    const { message } = App.useApp();
    const token = useUserStore((state) => state.token);
    const [summary, setSummary] = useState<AdminFinanceSummary | null>(null);
	const [readiness, setReadiness] = useState<AdminRuntimeReadiness | null>(null);
    const [expenses, setExpenses] = useState<AdminOperatingExpense[]>([]);
    const [expenseOpen, setExpenseOpen] = useState(false);
    const [expenseForm] = Form.useForm();
    const load = async () => { if (token) { const [finance, runtime, expenseItems] = await Promise.all([fetchAdminFinanceSummary(token), fetchAdminRuntimeReadiness(token), fetchAdminOperatingExpenses(token)]); setSummary(finance); setReadiness(runtime); setExpenses(expenseItems); } };
    useEffect(() => { void load(); }, [token]);
    const today = summary?.today;
    const allTime = summary?.allTime;
    return <main className="p-6"><Space direction="vertical" size={16} className="w-full">
		{readiness && !readiness.ready ? <Alert type="warning" showIcon message="平台尚未达到正式收款条件" description={<ul className="mb-0 pl-5">{readiness.issues.map((issue) => <li key={issue}>{issue}</li>)}</ul>} /> : null}
        <Alert type="info" showIcon message="预付资金、上游准备金和利润分开核算" description="未消费预付余额不计入利润；实际成本缺失的任务先按配置成本标记为 estimated，后续只能通过调整流水校正。" />
        <Alert type="info" showIcon message="提现不在画布里执行" description="用户付款会进入你配置的微信或支付宝商户账户；可提现金额、手续费和结算时间以对应商户后台为准。画布后台负责记录充值、消费、退款和用户未消费余额，不能把累计消费直接当作可提现利润。" action={<Button href="/admin/model-routing">调整模型售价</Button>} />
        <Typography.Title level={4} className="!mb-0">今日</Typography.Title>
        <Row gutter={[16,16]}>
            <Col xs={24} md={12} xl={6}><Card><Statistic title="充值到账" value={formatCNY(today?.rechargeCents || 0)} /></Card></Col>
            <Col xs={24} md={12} xl={6}><Card><Statistic title="生成结算收入" value={formatCNY(today?.revenueCents || 0)} /></Card></Col>
            <Col xs={24} md={12} xl={6}><Card><Statistic title="失败解冻" value={formatCNY(today?.releasedCents || 0)} /></Card></Col>
            <Col xs={24} md={12} xl={6}><Card><Statistic title="成功 / 失败任务" value={`${today?.settledTasks || 0} / ${today?.releasedTasks || 0}`} /></Card></Col>
        </Row>
		<Card title="运营费用" extra={<Button type="primary" onClick={() => setExpenseOpen(true)}>登记费用</Button>}>
			<Table rowKey="id" size="small" dataSource={expenses} pagination={{ pageSize: 10 }} columns={[
				{ title: "日期", dataIndex: "date" }, { title: "类别", dataIndex: "category" },
				{ title: "金额", render: (_: unknown, item: AdminOperatingExpense) => formatCNY(item.amountCny) },
				{ title: "原因", dataIndex: "reason" }, { title: "凭证", dataIndex: "reference", render: (value: string) => value || "-" },
			]} />
		</Card>
        <Typography.Title level={4} className="!mb-0">平台总览</Typography.Title>
        <Row gutter={[16,16]}>
            <Col xs={24} md={12} xl={6}><Card><Statistic title="用户数" value={summary?.userCount || 0} /></Card></Col>
            <Col xs={24} md={12} xl={6}><Card><Statistic title="用户可用余额" value={formatCNY(summary?.availableBalanceCents || 0)} /></Card></Col>
            <Col xs={24} md={12} xl={6}><Card><Statistic title="任务冻结余额" value={formatCNY(summary?.frozenBalanceCents || 0)} /></Card></Col>
            <Col xs={24} md={12} xl={6}><Card><Statistic title="累计充值到账" value={formatCNY(allTime?.rechargeCents || 0)} /></Card></Col>
            <Col xs={24} md={12} xl={6}><Card><Statistic title="累计生成收入" value={formatCNY(allTime?.revenueCents || 0)} /></Card></Col>
            <Col xs={24} md={12} xl={6}><Card><Statistic title="累计失败解冻" value={formatCNY(allTime?.releasedCents || 0)} /></Card></Col>
            <Col xs={24} md={12} xl={6}><Card><Statistic title="未消费预付余额" value={formatCNY(summary?.unconsumedBalanceCents || 0)} /></Card></Col>
            <Col xs={24} md={12} xl={6}><Card><Statistic title="实际上游成本" value={formatCNY(summary?.actualProviderCostCents || 0)} /></Card></Col>
            <Col xs={24} md={12} xl={6}><Card><Statistic title="上游成本准备金" value={formatCNY(summary?.providerReserveCents || 0)} /></Card></Col>
            <Col xs={24} md={12} xl={6}><Card><Statistic title="已实现模型毛差" value={formatCNY(summary?.grossProfitCents || 0)} /></Card></Col>
            <Col xs={24} md={12} xl={6}><Card><Statistic title="支付手续费" value={formatCNY(summary?.paymentFeeCents || 0)} /></Card></Col>
            <Col xs={24} md={12} xl={6}><Card><Statistic title="补偿" value={formatCNY(summary?.compensationCents || 0)} /></Card></Col>
            <Col xs={24} md={12} xl={6}><Card><Statistic title="运营费用" value={formatCNY(summary?.operatingCostCents || 0)} /></Card></Col>
            <Col xs={24} md={12} xl={6}><Card><Statistic title="估算净收益" value={formatCNY(summary?.estimatedNetProfitCents || 0)} /></Card></Col>
        </Row>
		<Typography.Title level={4} className="!mb-0">上线检查</Typography.Title>
		<Row gutter={[16,16]}>
			<Col xs={24} md={12} xl={6}><Card><Statistic title="数据库" value={readiness?.databasePersistent ? "持久化" : "临时"} valueStyle={{ color: readiness?.databasePersistent ? "#3f8600" : "#cf1322" }} /><Typography.Text type="secondary">驱动：{readiness?.databaseDriver || "检查中"}</Typography.Text></Card></Col>
			<Col xs={24} md={12} xl={6}><Card><Statistic title="微信 / 支付宝" value={readiness?.paymentConfigured ? "已配置" : "未配置"} valueStyle={{ color: readiness?.paymentConfigured ? "#3f8600" : "#cf1322" }} /></Card></Col>
			<Col xs={24} md={12} xl={6}><Card><Statistic title="作品对象存储" value={readiness?.storageConfigured ? "已配置" : "未配置"} valueStyle={{ color: readiness?.storageConfigured ? "#3f8600" : "#cf1322" }} /></Card></Col>
			<Col xs={24} md={12} xl={6}><Card><Statistic title="平台托管模型" value={readiness?.managedPlatform ? "已开启" : "未开启"} valueStyle={{ color: readiness?.managedPlatform ? "#3f8600" : "#cf1322" }} /></Card></Col>
		</Row>
        <Modal title="登记运营费用" open={expenseOpen} onCancel={() => setExpenseOpen(false)} onOk={async () => { const values = await expenseForm.validateFields(); await recordAdminOperatingExpense(token, { category: values.category, amountCents: Math.round(values.amountYuan * 100), date: values.date?.format("YYYY-MM-DD"), reason: values.reason, reference: values.reference }); message.success("运营费用已登记"); setExpenseOpen(false); expenseForm.resetFields(); await load(); }}>
            <Form form={expenseForm} layout="vertical">
                <Form.Item name="category" label="类别" rules={[{ required: true }]}><Select options={["server", "database", "storage", "cdn", "domain", "payment_fee", "other"].map((value) => ({ value, label: value }))} /></Form.Item>
                <Form.Item name="amountYuan" label="金额（人民币元）" rules={[{ required: true }]}><InputNumber min={0.01} precision={2} className="w-full" /></Form.Item>
                <Form.Item name="date" label="费用日期"><DatePicker className="w-full" /></Form.Item>
                <Form.Item name="reason" label="原因" rules={[{ required: true, whitespace: true }]}><Input /></Form.Item>
                <Form.Item name="reference" label="凭证 / 参考"><Input /></Form.Item>
            </Form>
        </Modal>
    </Space></main>;
}
