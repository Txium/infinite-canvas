"use client";

import { Alert, App, Button, Card, Col, DatePicker, Form, Input, InputNumber, Modal, Row, Segmented, Select, Space, Statistic, Table, Tag, Typography } from "antd";
import { useEffect, useState } from "react";

import { formatCNY } from "@/constant/credits";
import { fetchAdminFinanceSummary, fetchAdminOperatingExpenses, fetchAdminRuntimeReadiness, recordAdminOperatingExpense, type AdminFinanceSummary, type AdminOperatingExpense, type AdminRuntimeReadiness, type ModelProfitSummary, type ProviderCostSummary } from "@/services/api/admin";
import { useUserStore } from "@/stores/use-user-store";

export default function AdminFinancePage() {
    const { message } = App.useApp();
    const token = useUserStore((state) => state.token);
    const [summary, setSummary] = useState<AdminFinanceSummary | null>(null);
	const [readiness, setReadiness] = useState<AdminRuntimeReadiness | null>(null);
    const [expenses, setExpenses] = useState<AdminOperatingExpense[]>([]);
    const [expenseOpen, setExpenseOpen] = useState(false);
	const [period, setPeriod] = useState("all");
    const [expenseForm] = Form.useForm();
    const load = async () => { if (token) { const [finance, runtime, expenseItems] = await Promise.all([fetchAdminFinanceSummary(token, period), fetchAdminRuntimeReadiness(token), fetchAdminOperatingExpenses(token)]); setSummary(finance); setReadiness(runtime); setExpenses(expenseItems); } };
    useEffect(() => { void load(); }, [token, period]);
    const allTime = summary?.allTime;
    return <main className="p-6"><Space direction="vertical" size={16} className="w-full">
		{readiness && !readiness.ready ? <Alert type="warning" showIcon message="平台尚未达到正式收款条件" description={<ul className="mb-0 pl-5">{readiness.issues.map((issue) => <li key={issue}>{issue}</li>)}</ul>} /> : null}
        <Alert type="info" showIcon message="预付资金、上游准备金和利润分开核算" description="未消费预付余额不计入利润；实际成本缺失的任务先按配置成本标记为 estimated，后续只能通过调整流水校正。" />
        {summary && !summary.upstreamCostReady ? <Alert type="warning" showIcon message="历史上游成本不完整" description="当前已有生成收入，但部分历史成功任务没有 Provider 成本流水；上游成本、毛差和净收益只能视为待补账数据，不能作为真实利润。" /> : null}
        <Alert type="info" showIcon message="提现不在画布里执行" description="用户付款会进入你配置的微信或支付宝商户账户；可提现金额、手续费和结算时间以对应商户后台为准。画布后台负责记录充值、消费、退款和用户未消费余额，不能把累计消费直接当作可提现利润。" action={<Button href="/admin/model-routing">调整模型售价</Button>} />
        <Space className="w-full justify-between"><Typography.Title level={4} className="!mb-0">资金总览</Typography.Title><Segmented value={period} onChange={(value) => setPeriod(String(value))} options={[{label:"今日",value:"today"},{label:"昨日",value:"yesterday"},{label:"7天",value:"7d"},{label:"30天",value:"30d"},{label:"累计",value:"all"}]} /></Space>
        <Row gutter={[16,16]}>
            <Col xs={24} md={12} xl={6}><Card><Statistic title="累计实际充值" value={formatCNY(summary?.selected?.rechargeCents || 0)} /></Card></Col>
            <Col xs={24} md={12} xl={6}><Card><Statistic title="用户未消费余额" value={formatCNY(summary?.unconsumedBalanceCents || 0)} /></Card></Col>
            <Col xs={24} md={12} xl={6}><Card><Statistic title="冻结余额" value={formatCNY(summary?.frozenBalanceCents || 0)} /></Card></Col>
            <Col xs={24} md={12} xl={6}><Card><Statistic title="累计生成收入" value={formatCNY(summary?.selected?.revenueCents || 0)} /></Card></Col>
            <Col xs={24} md={12} xl={6}><Card><Statistic title="实际上游成本" value={formatCNY(summary?.selectedProviderCostCents || 0)} /></Card></Col>
            <Col xs={24} md={12} xl={6}><Card><Statistic title="Provider Reserve" value={formatCNY(summary?.providerReserveCents || 0)} /></Card></Col>
            <Col xs={24} md={12} xl={6}><Card><Statistic title="已实现毛差" value={formatCNY(summary?.selectedGrossProfitCents || 0)} /></Card></Col>
            <Col xs={24} md={12} xl={6}><Card><Statistic title="支付手续费" value={formatCNY(summary?.selectedPaymentFeeCents || 0)} /></Card></Col>
            <Col xs={24} md={12} xl={6}><Card><Statistic title="补偿 / 退款成本" value={formatCNY(summary?.selectedCompensationCents || 0)} /></Card></Col>
            <Col xs={24} md={12} xl={6}><Card><Statistic title="运营费用" value={formatCNY(summary?.selectedOperatingCostCents || 0)} /></Card></Col>
            <Col xs={24} md={12} xl={6}><Card><Statistic title="估算净收益" value={formatCNY(summary?.selectedNetProfitCents || 0)} /></Card></Col>
            <Col xs={24} md={12} xl={6}><Card><Statistic title="失败解冻" value={formatCNY(summary?.selected?.releasedCents || 0)} /></Card></Col>
        </Row>
		<Card title="模型利润排行"><Table rowKey="model" size="small" pagination={false} dataSource={summary?.modelProfits || []} columns={[{title:"模型 / 档位",dataIndex:"model"},{title:"生成次数",dataIndex:"taskCount"},{title:"生成收入",render:(_:unknown,item:ModelProfitSummary)=>formatCNY(item.revenueCents)},{title:"上游成本",render:(_:unknown,item:ModelProfitSummary)=><Space>{formatCNY(item.providerCostCents)}{item.unconfirmedCostTaskCount>0?<Tag color="red">成本缺失</Tag>:item.estimatedCostTaskCount>0?<Tag color="orange">含预估</Tag>:<Tag color="green">实际</Tag>}</Space>},{title:"毛差",render:(_:unknown,item:ModelProfitSummary)=>item.unconfirmedCostTaskCount>0?"待补账":formatCNY(item.grossProfitCents)},{title:"毛利率",render:(_:unknown,item:ModelProfitSummary)=>item.unconfirmedCostTaskCount>0?"-":item.revenueCents?`${Math.round(item.grossProfitCents*10000/item.revenueCents)/100}%`:"-"}]} /></Card>
		<Card title="Provider 成本统计"><Table rowKey="provider" size="small" pagination={false} dataSource={summary?.providerCosts || []} columns={[{title:"中转站",dataIndex:"provider",render:(value:string)=>({"302":"302.AI",wavespeed:"WaveSpeed",lec:"LEC",seedance_nz:"seedance.nz"}[value]||value)},{title:"今日实际成本",render:(_:unknown,item:ProviderCostSummary)=>formatCNY(item.todayCents)},{title:"近7日实际成本",render:(_:unknown,item:ProviderCostSummary)=>formatCNY(item.last7DaysCents)},{title:"累计实际成本",render:(_:unknown,item:ProviderCostSummary)=>formatCNY(item.allTimeCents)}]} /></Card>
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
