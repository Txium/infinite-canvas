"use client";

import { App, Button, Card, Form, Input, InputNumber, Modal, Select, Space, Switch, Table, Tabs, Tag, Typography } from "antd";
import { useEffect, useState } from "react";

import { fetchAdminMarketModels, fetchAdminModelProviders, fetchAdminModelReadiness, fetchAdminModelRoutes, fetchAdminModelVariants, saveAdminMarketModel, saveAdminModelRoute, saveAdminModelVariant, type AdminMarketModel, type AdminModelProvider, type AdminModelReadiness, type AdminModelRoute, type AdminModelVariant } from "@/services/api/admin";
import { useUserStore } from "@/stores/use-user-store";

type VariantForm = Omit<AdminModelVariant, "priceCents"> & { priceYuan?: number };

const categoryOptions = [
    { label: "大语言模型", value: "llm" }, { label: "图片", value: "image" }, { label: "视频", value: "video" },
    { label: "人物 / 数字人", value: "person" }, { label: "音乐", value: "music" }, { label: "语音", value: "voice" },
    { label: "3D", value: "3d" }, { label: "AI 工具", value: "tool" },
];

export default function AdminModelRoutingPage() {
    const { message } = App.useApp();
    const token = useUserStore((state) => state.token);
    const [providers, setProviders] = useState<AdminModelProvider[]>([]);
    const [models, setModels] = useState<AdminMarketModel[]>([]);
    const [routes, setRoutes] = useState<AdminModelRoute[]>([]);
    const [variants, setVariants] = useState<AdminModelVariant[]>([]);
    const [editingModel, setEditingModel] = useState<AdminMarketModel | null>(null);
    const [routeOpen, setRouteOpen] = useState(false);
    const [variantOpen, setVariantOpen] = useState(false);
    const [readiness, setReadiness] = useState<AdminModelReadiness | null>(null);
    const [modelForm] = Form.useForm<AdminMarketModel>();
    const [routeForm] = Form.useForm<AdminModelRoute>();
    const [variantForm] = Form.useForm<VariantForm>();
    const load = async () => {
        if (!token) return;
        const results = await Promise.allSettled([fetchAdminModelProviders(token), fetchAdminMarketModels(token), fetchAdminModelRoutes(token), fetchAdminModelVariants(token)]);
        const labels = ["中转站", "模型", "线路", "档位"];
        results.forEach((result, index) => {
            if (result.status === "rejected") message.error(`${labels[index]}资料加载失败，请稍后重试`);
        });
        if (results[0].status === "fulfilled") setProviders(results[0].value);
        if (results[1].status === "fulfilled") setModels(results[1].value);
        if (results[2].status === "fulfilled") setRoutes(results[2].value);
        if (results[3].status === "fulfilled") setVariants(results[3].value);
    };
    useEffect(() => { void load(); }, [token]);
    const saveModel = async () => {
        if (!editingModel || !token) return;
        const values = await modelForm.validateFields();
        await saveAdminMarketModel(token, { ...editingModel, ...values });
        message.success("模型资料已保存，模型广场会自动更新"); setEditingModel(null); await load();
    };
    const saveRoute = async () => {
        const values = await routeForm.validateFields(); const variant = variants.find((item) => item.id === values.variantId);
        await saveAdminModelRoute(token!, { ...values, modelId: variant?.modelId || values.modelId });
        message.success("模型线路已保存"); setRouteOpen(false); routeForm.resetFields(); await load();
    };
    const saveVariant = async () => {
        const values = await variantForm.validateFields(); const { priceYuan, ...variantValues } = values;
        await saveAdminModelVariant(token!, { ...variantValues, priceCents: priceYuan == null ? undefined : Math.round(priceYuan * 100) });
        message.success("模型档位已保存，模型广场会自动更新"); setVariantOpen(false); variantForm.resetFields(); await load();
    };
    const checkReadiness = async () => { if (token) setReadiness(await fetchAdminModelReadiness(token)); };
    return <div className="p-6"><Card title="模型、档位与上游线路" extra={<Space><Button onClick={() => void load()}>刷新</Button><Button onClick={() => void checkReadiness()}>上线检查</Button><Button href="/admin/providers">中转站管理</Button><Button type="primary" onClick={() => setRouteOpen(true)}>配置档位线路</Button></Space>}>
        <Tabs items={[
            { key: "models", label: `模型管理 ${models.length}`, children: <Table rowKey="id" dataSource={models} pagination={{ pageSize: 20 }} columns={[
                { title: "模型", render: (_: unknown, item: AdminMarketModel) => <Space direction="vertical" size={0}><Typography.Text strong>{item.name}</Typography.Text><Typography.Text type="secondary">{item.id}</Typography.Text></Space> },
                { title: "分类", dataIndex: "category" },
                { title: "档位数", render: (_: unknown, item: AdminMarketModel) => variants.filter((variant) => variant.modelId === item.id).length },
                { title: "热门", dataIndex: "featured", render: (value: boolean) => <Tag color={value ? "gold" : "default"}>{value ? "热门" : "普通"}</Tag> },
                { title: "运行状态", dataIndex: "status", render: (value: AdminMarketModel["status"]) => <Tag color={value === "normal" ? "success" : value === "busy" ? "warning" : "error"}>{value === "normal" ? "正常" : value === "busy" ? "拥堵" : "维护"}</Tag> },
                { title: "上架", dataIndex: "enabled", render: (value: boolean) => <Tag color={value ? "success" : "default"}>{value ? "展示" : "隐藏"}</Tag> },
                { title: "操作", render: (_: unknown, item: AdminMarketModel) => <Button size="small" onClick={() => { setEditingModel(item); modelForm.setFieldsValue(item); }}>编辑模型</Button> },
            ]} /> },
            { key: "variants", label: `档位与售价 ${variants.length}`, children: <Table rowKey="id" dataSource={variants} pagination={{ pageSize: 20 }} columns={[
                { title: "模型", dataIndex: "modelId", render: (value: string) => models.find((item) => item.id === value)?.name || value },
                { title: "档位", dataIndex: "name" },
                { title: "售价", render: (_: unknown, item: AdminModelVariant) => item.pricingMode === "fixed" && typeof item.priceCents === "number" ? `¥${(item.priceCents / 100).toFixed(2)} ${item.billingUnit}` : item.pricingMode === "dynamic" ? item.priceFormula : "暂不上架" },
                { title: "状态", dataIndex: "enabled", render: (value: boolean) => <Tag color={value ? "success" : "default"}>{value ? "上架" : "下架"}</Tag> },
                { title: "操作", render: (_: unknown, item: AdminModelVariant) => <Button size="small" onClick={() => { variantForm.setFieldsValue({ ...item, priceYuan: item.priceCents == null ? undefined : item.priceCents / 100 }); setVariantOpen(true); }}>修改售价 / 状态</Button> },
            ]} /> },
            { key: "routes", label: `线路 ${routes.length}`, children: <Table rowKey="id" dataSource={routes} pagination={{ pageSize: 20 }} columns={[
                { title: "模型档位", dataIndex: "variantId", render: (value: string) => { const item = variants.find((variant) => variant.id === value); return item ? `${models.find((model) => model.id === item.modelId)?.name || item.modelId} / ${item.name}` : value; } },
                { title: "供应商", dataIndex: "providerId", render: (value: string) => providers.find((item) => item.id === value)?.name || value },
                { title: "实际上游 ID", dataIndex: "upstreamModelId" }, { title: "顺序", dataIndex: "priority" },
                { title: "状态", dataIndex: "enabled", render: (value: boolean) => <Tag color={value ? "success" : "default"}>{value ? "启用" : "停用"}</Tag> },
                { title: "操作", render: (_: unknown, item: AdminModelRoute) => <Button size="small" onClick={() => { routeForm.setFieldsValue(item); setRouteOpen(true); }}>编辑</Button> },
            ]} /> },
        ]} />
    </Card>
    <Modal title={editingModel ? `编辑 ${editingModel.name}` : "编辑模型"} open={!!editingModel} onCancel={() => setEditingModel(null)} onOk={() => void saveModel()} width={680}><Form form={modelForm} layout="vertical"><Form.Item name="name" label="前台名称" rules={[{ required: true }]}><Input /></Form.Item><Form.Item name="category" label="分类" rules={[{ required: true }]}><Select options={categoryOptions} /></Form.Item><Form.Item name="description" label="模型描述"><Input.TextArea rows={3} /></Form.Item><Space align="start" size={16}><Form.Item name="sort" label="排序"><InputNumber min={0} precision={0} /></Form.Item><Form.Item name="status" label="运行状态"><Select style={{ width: 120 }} options={[{ label: "正常", value: "normal" }, { label: "拥堵", value: "busy" }, { label: "维护", value: "maintenance" }]} /></Form.Item><Form.Item name="featured" label="热门推荐" valuePropName="checked"><Switch /></Form.Item><Form.Item name="enabled" label="模型广场展示" valuePropName="checked"><Switch /></Form.Item></Space></Form></Modal>
    <Modal title="模型档位售价" open={variantOpen} onCancel={() => setVariantOpen(false)} onOk={() => void saveVariant()}><Form form={variantForm} layout="vertical"><Form.Item name="id" hidden><Input /></Form.Item><Form.Item name="modelId" hidden><Input /></Form.Item><Form.Item name="name" label="档位"><Input disabled /></Form.Item><Form.Item name="pricingMode" label="定价方式"><Select options={[{ label: "固定人民币", value: "fixed" }, { label: "实际成本动态加价", value: "dynamic" }, { label: "暂不上架", value: "disabled" }]} /></Form.Item><Form.Item name="priceYuan" label="用户售价（人民币）"><InputNumber min={0} precision={2} className="w-full" prefix="¥" /></Form.Item><Form.Item name="priceFormula" label="动态定价公式"><Input placeholder="例如：实际成本×1.08" /></Form.Item><Form.Item name="billingUnit" label="计费单位"><Input /></Form.Item><Form.Item name="enabled" label="上架" valuePropName="checked"><Switch /></Form.Item></Form></Modal>
    <Modal title="模型档位线路" open={routeOpen} onCancel={() => setRouteOpen(false)} onOk={() => void saveRoute()}><Form form={routeForm} layout="vertical" initialValues={{ enabled: false, priority: 1, protocol: "custom" }}><Form.Item name="id" hidden><Input /></Form.Item><Form.Item name="modelId" hidden><Input /></Form.Item><Form.Item name="variantId" label="模型档位" rules={[{ required: true }]}><Select showSearch options={variants.map((item) => ({ label: `${models.find((model) => model.id === item.modelId)?.name || item.modelId} / ${item.name}`, value: item.id }))} /></Form.Item><Form.Item name="providerId" label="供应商" rules={[{ required: true }]}><Select options={providers.map((item) => ({ label: item.name, value: item.id }))} /></Form.Item><Form.Item name="upstreamModelId" label="实际上游 ID" rules={[{ required: true }]}><Input /></Form.Item><Form.Item name="protocol" label="协议"><Select options={[{ label: "OpenAI 兼容", value: "openai" }, { label: "WaveSpeed", value: "wavespeed" }, { label: "自定义", value: "custom" }]} /></Form.Item><Form.Item name="priority" label="线路顺序"><InputNumber min={1} /></Form.Item><Form.Item name="enabled" label="启用" valuePropName="checked"><Switch /></Form.Item></Form></Modal>
    <Modal title="模型系统上线检查" open={!!readiness} onCancel={() => setReadiness(null)} footer={<Button type="primary" onClick={() => setReadiness(null)}>知道了</Button>}><Space direction="vertical" className="w-full"><Tag color={readiness?.ready ? "success" : "error"}>{readiness?.ready ? "已有可生成档位" : "尚不能真实生成"}</Tag><Typography.Text>供应商：{readiness?.readyProviderCount || 0} / {readiness?.providerCount || 0} 可用</Typography.Text><Typography.Text>模型档位：{readiness?.availableVariantCount || 0} / {readiness?.enabledVariantCount || 0} 可生成</Typography.Text><Typography.Text>启用线路：{readiness?.enabledRouteCount || 0}</Typography.Text>{readiness?.issues.map((item, index) => <Typography.Text key={`${item.scope}-${item.id}-${index}`} type={item.level === "error" ? "danger" : "warning"}>• {item.message}</Typography.Text>)}</Space></Modal>
    </div>;
}
