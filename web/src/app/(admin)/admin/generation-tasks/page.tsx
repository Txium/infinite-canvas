"use client";

import { ReloadOutlined } from "@ant-design/icons";
import { ProTable, type ProColumns } from "@ant-design/pro-components";
import { Button, Card, Space, Tag, Typography } from "antd";
import dayjs from "dayjs";
import { useEffect, useState } from "react";

import { formatCNY } from "@/constant/credits";
import { fetchAdminGenerationTasks, type AdminGenerationTask } from "@/services/api/admin";
import { useUserStore } from "@/stores/use-user-store";

const kindLabels: Record<string, string> = { image: "图片", video: "视频", audio: "音频" };
const billingLabels: Record<string,string> = { frozen:"已冻结", settled:"已结算", released:"已解冻" };

export default function AdminGenerationTasksPage() {
    const token = useUserStore((state) => state.token);
    const [items,setItems] = useState<AdminGenerationTask[]>([]);
    const [loading,setLoading] = useState(false);
    const load = async () => { if (!token) return; setLoading(true); try { setItems(await fetchAdminGenerationTasks(token)); } finally { setLoading(false); } };
    useEffect(() => { void load(); }, [token]);
    const columns: ProColumns<AdminGenerationTask>[] = [
        {title:"用户",dataIndex:"userDisplayName",width:160,render:(_,item)=><Space direction="vertical" size={0}><Typography.Text>{item.userDisplayName||item.userId}</Typography.Text><Typography.Text type="secondary" copyable>{item.userId}</Typography.Text></Space>},
        {title:"类型",dataIndex:"kind",width:80,render:(_,item)=><Tag>{kindLabels[item.kind]}</Tag>},
        {title:"模型档位",dataIndex:"model",width:220,ellipsis:true},
        {title:"任务状态",dataIndex:"status",width:110,render:(_,item)=><Tag color={item.status==="completed"?"success":item.status==="failed"?"error":"processing"}>{item.status}</Tag>},
        {title:"资金状态",dataIndex:"billingStatus",width:110,render:(_,item)=><Tag color={item.billingStatus==="settled"?"success":item.billingStatus==="released"?"default":"warning"}>{billingLabels[item.billingStatus]||"未计费"}</Tag>},
        {title:"售价",dataIndex:"priceCents",width:100,render:(_,item)=>formatCNY(item.priceCents)},
        {title:"结果",dataIndex:"resultUrl",width:90,render:(_,item)=>item.resultUrl?<Typography.Link href={item.resultUrl} target="_blank" rel="noreferrer">查看</Typography.Link>:"-"},
        {title:"错误",dataIndex:"error",ellipsis:true,render:(_,item)=><Typography.Text type={item.error?"danger":"secondary"}>{item.error||"-"}</Typography.Text>},
        {title:"创建时间",dataIndex:"createdAt",width:180,render:(_,item)=>item.createdAt?dayjs(item.createdAt).format("YYYY-MM-DD HH:mm:ss"):"-"},
    ];
    return <main style={{padding:24}}><Card variant="borderless"><ProTable rowKey="id" columns={columns} dataSource={items} loading={loading} search={false} pagination={{pageSize:20}} headerTitle={<Space><Typography.Text strong>生成任务</Typography.Text><Tag>{items.length} 条</Tag></Space>} toolBarRender={()=>[<Button key="refresh" icon={<ReloadOutlined/>} onClick={()=>void load()}>刷新</Button>]} /></Card></main>;
}
