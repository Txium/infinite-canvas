"use client";

import { App, Button, Card, Form, Input, InputNumber, Modal, Select, Space, Switch, Table, Tabs, Tag } from "antd";
import { useEffect, useState } from "react";

import { fetchAdminModelProviders, fetchAdminModelRoutes, saveAdminModelProvider, saveAdminModelRoute, type AdminModelProvider, type AdminModelRoute } from "@/services/api/admin";
import { fetchModelMarket, type MarketModelCard } from "@/services/api/model-market";
import { useUserStore } from "@/stores/use-user-store";

export default function AdminModelRoutingPage() {
    const { message } = App.useApp();
    const token = useUserStore((state) => state.token);
    const [providers, setProviders] = useState<AdminModelProvider[]>([]);
    const [models, setModels] = useState<MarketModelCard[]>([]);
    const [routes, setRoutes] = useState<AdminModelRoute[]>([]);
    const [providerOpen, setProviderOpen] = useState(false);
    const [routeOpen, setRouteOpen] = useState(false);
    const [providerForm] = Form.useForm<AdminModelProvider>();
    const [routeForm] = Form.useForm<AdminModelRoute>();
    const load = async () => { if (!token) return; const [nextProviders, nextModels, nextRoutes] = await Promise.all([fetchAdminModelProviders(token), fetchModelMarket(), fetchAdminModelRoutes(token)]); setProviders(nextProviders); setModels(nextModels); setRoutes(nextRoutes); };
    useEffect(() => { void load(); }, [token]);
    const saveProvider = async () => { const values = await providerForm.validateFields(); await saveAdminModelProvider(token!, values); message.success("供应商已保存"); setProviderOpen(false); providerForm.resetFields(); await load(); };
    const saveRoute = async () => { const values = await routeForm.validateFields(); await saveAdminModelRoute(token!, values); message.success("模型线路已保存"); setRouteOpen(false); routeForm.resetFields(); await load(); };
    return <div className="p-6"><Card title="Providers + Models + Routes" extra={<Space><Button onClick={() => void load()}>刷新</Button><Button onClick={() => setProviderOpen(true)}>新增供应商</Button><Button type="primary" onClick={() => setRouteOpen(true)}>新增线路</Button></Space>}>
        <Tabs items={[
            { key: "models", label: `模型 ${models.length}`, children: <Table rowKey="id" dataSource={models} pagination={{ pageSize: 20 }} columns={[{ title: "自有 ID", dataIndex: "id" }, { title: "名称", dataIndex: "name" }, { title: "分类", dataIndex: "category" }, { title: "状态", render: (_: unknown, item: MarketModelCard) => <Tag color={item.available ? "success" : "default"}>{item.available ? "已接线路" : "待接入"}</Tag> }]} /> },
            { key: "providers", label: `供应商 ${providers.length}`, children: <Table rowKey="id" dataSource={providers} columns={[{ title: "名称", dataIndex: "name" }, { title: "代码", dataIndex: "code" }, { title: "Base URL", dataIndex: "baseUrl" }, { title: "状态", dataIndex: "enabled", render: (value: boolean) => <Tag color={value ? "success" : "default"}>{value ? "启用" : "停用"}</Tag> }, { title: "操作", render: (_: unknown, item: AdminModelProvider) => <Button size="small" onClick={() => { providerForm.setFieldsValue(item); setProviderOpen(true); }}>编辑</Button> }]} /> },
            { key: "routes", label: `线路 ${routes.length}`, children: <Table rowKey="id" dataSource={routes} columns={[{ title: "自有模型", dataIndex: "modelId" }, { title: "供应商", dataIndex: "providerId", render: (value: string) => providers.find((item) => item.id === value)?.name || value }, { title: "实际上游 ID", dataIndex: "upstreamModelId" }, { title: "顺序", dataIndex: "priority" }, { title: "状态", dataIndex: "enabled", render: (value: boolean) => <Tag color={value ? "success" : "default"}>{value ? "启用" : "停用"}</Tag> }, { title: "操作", render: (_: unknown, item: AdminModelRoute) => <Button size="small" onClick={() => { routeForm.setFieldsValue(item); setRouteOpen(true); }}>编辑</Button> }]} /> },
        ]} />
    </Card>
    <Modal title="供应商" open={providerOpen} onCancel={() => setProviderOpen(false)} onOk={() => void saveProvider()}><Form form={providerForm} layout="vertical" initialValues={{ enabled: true, priority: 10, timeout: 300 }}><Form.Item name="id" hidden><Input /></Form.Item><Form.Item name="name" label="名称" rules={[{ required: true }]}><Input placeholder="WaveSpeed" /></Form.Item><Form.Item name="code" label="代码" rules={[{ required: true }]}><Input placeholder="wavespeed" /></Form.Item><Form.Item name="baseUrl" label="Base URL" rules={[{ required: true }]}><Input /></Form.Item><Form.Item name="apiKey" label="API Key" extra="编辑时留空会保留原密钥"><Input.Password /></Form.Item><Form.Item name="priority" label="优先级"><InputNumber min={1} /></Form.Item><Form.Item name="timeout" label="超时秒数"><InputNumber min={30} /></Form.Item><Form.Item name="enabled" label="启用" valuePropName="checked"><Switch /></Form.Item></Form></Modal>
    <Modal title="模型线路" open={routeOpen} onCancel={() => setRouteOpen(false)} onOk={() => void saveRoute()}><Form form={routeForm} layout="vertical" initialValues={{ enabled: true, priority: 1, protocol: "openai" }}><Form.Item name="id" hidden><Input /></Form.Item><Form.Item name="modelId" label="自有模型" rules={[{ required: true }]}><Select showSearch options={models.map((item) => ({ label: `${item.name} (${item.id})`, value: item.id }))} /></Form.Item><Form.Item name="providerId" label="供应商" rules={[{ required: true }]}><Select options={providers.map((item) => ({ label: item.name, value: item.id }))} /></Form.Item><Form.Item name="upstreamModelId" label="实际上游 ID" rules={[{ required: true }]}><Input /></Form.Item><Form.Item name="protocol" label="协议"><Select options={[{ label: "OpenAI 兼容", value: "openai" }, { label: "WaveSpeed", value: "wavespeed" }, { label: "自定义", value: "custom" }]} /></Form.Item><Form.Item name="priority" label="线路顺序"><InputNumber min={1} /></Form.Item><Form.Item name="enabled" label="启用" valuePropName="checked"><Switch /></Form.Item></Form></Modal>
    </div>;
}
