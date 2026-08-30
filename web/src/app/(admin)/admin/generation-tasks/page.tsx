"use client";

import { ReloadOutlined } from "@ant-design/icons";
import { ProTable, type ActionType, type ProColumns } from "@ant-design/pro-components";
import { Button, Card, Image, Space, Tag, Typography } from "antd";
import dayjs from "dayjs";
import { useRef } from "react";

import { formatCNY } from "@/constant/credits";
import { fetchAdminGenerationTasks, type AdminGenerationTask } from "@/services/api/admin";
import { useUserStore } from "@/stores/use-user-store";

const kindLabels: Record<string, string> = { image: "图片", video: "视频", audio: "音频" };
const billingLabels: Record<string,string> = { frozen:"已冻结", settled:"已结算", released:"已解冻" };

export default function AdminGenerationTasksPage() {
    const token = useUserStore((state) => state.token);
    const actionRef = useRef<ActionType>(null);
    const columns: ProColumns<AdminGenerationTask>[] = [
        {title:"搜索",dataIndex:"keyword",hideInTable:true},
        {title:"用户",dataIndex:"userDisplayName",width:160,search:false,render:(_,item)=><Space direction="vertical" size={0}><Typography.Text>{item.userDisplayName||item.userId}</Typography.Text><Typography.Text type="secondary" copyable>{item.userId}</Typography.Text></Space>},
        {title:"类型",dataIndex:"kind",width:80,valueType:"select",valueEnum:{image:{text:"图片"},video:{text:"视频"},audio:{text:"音频"}},render:(_,item)=><Tag>{kindLabels[item.kind]}</Tag>},
        {title:"模型档位",dataIndex:"model",width:220,ellipsis:true,search:false},
		{title:"中转站",dataIndex:"channelName",width:130,ellipsis:true,search:false,render:(_,item)=>item.channelName||"-"},
		{title:"上游任务",dataIndex:"upstreamTaskId",width:150,ellipsis:true,search:false,render:(_,item)=>item.upstreamTaskId?<Typography.Text copyable>{item.upstreamTaskId}</Typography.Text>:"-"},
        {title:"任务状态",dataIndex:"status",width:110,valueType:"select",valueEnum:{queued:{text:"排队"},processing:{text:"处理中"},completed:{text:"成功"},failed:{text:"失败"}},render:(_,item)=><Tag color={item.status==="completed"?"success":item.status==="failed"?"error":"processing"}>{item.status}</Tag>},
        {title:"资金状态",dataIndex:"billingStatus",width:110,valueType:"select",valueEnum:{frozen:{text:"已冻结"},settled:{text:"已结算"},released:{text:"已解冻"}},render:(_,item)=><Tag color={item.billingStatus==="settled"?"success":item.billingStatus==="released"?"default":"warning"}>{billingLabels[item.billingStatus]||"未计费"}</Tag>},
        {title:"售价",dataIndex:"priceCents",width:100,render:(_,item)=>formatCNY(item.priceCents)},
		{title:"结果预览",dataIndex:"resultUrl",width:190,search:false,render:(_,item)=>!item.resultUrl?"-":item.kind==="image"?<Image src={item.resultUrl} width={120} height={80} style={{objectFit:"cover",borderRadius:8}}/>:item.kind==="video"?<video src={item.resultUrl} controls preload="metadata" style={{width:160,maxHeight:100,borderRadius:8,background:"#000"}}/>:<audio src={item.resultUrl} controls preload="metadata" style={{width:180}}/>},
        {title:"错误",dataIndex:"error",ellipsis:true,render:(_,item)=><Typography.Text type={item.error?"danger":"secondary"}>{item.error||"-"}</Typography.Text>},
        {title:"创建时间",dataIndex:"createdAt",width:180,search:false,render:(_,item)=>item.createdAt?dayjs(item.createdAt).format("YYYY-MM-DD HH:mm:ss"):"-"},
        {title:"创建日期",dataIndex:"createdRange",valueType:"dateRange",hideInTable:true},
    ];
    return <main style={{padding:24}}><Card variant="borderless"><ProTable actionRef={actionRef} rowKey="id" columns={columns} request={async(params)=>{ if(!token)return{data:[],success:true,total:0}; const range=params.createdRange as string[]|undefined; const data=await fetchAdminGenerationTasks(token,{keyword:params.keyword as string,kind:params.kind as string,status:params.status as string,billingStatus:params.billingStatus as string,startedAt:range?.[0],endedAt:range?.[1],limit:500}); return{data,success:true,total:data.length}; }} pagination={{pageSize:20}} headerTitle={<Typography.Text strong>生成任务</Typography.Text>} toolBarRender={()=>[<Button key="refresh" icon={<ReloadOutlined/>} onClick={()=>actionRef.current?.reload()}>刷新</Button>]} /></Card></main>;
}
