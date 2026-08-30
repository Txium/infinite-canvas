"use client";

import { ReloadOutlined } from "@ant-design/icons";
import { ProTable, type ActionType, type ProColumns } from "@ant-design/pro-components";
import { Button, Card, Form, Image, Input, Modal, Select, Space, Tag, Typography, message } from "antd";
import dayjs from "dayjs";
import { useRef, useState } from "react";

import { formatCNY } from "@/constant/credits";
import { fetchAdminGenerationTasks, importAdminGenerationTask, type AdminGenerationTask } from "@/services/api/admin";
import { useUserStore } from "@/stores/use-user-store";

const kindLabels: Record<string, string> = { image: "图片", video: "视频", audio: "音频" };
const billingLabels: Record<string,string> = { frozen:"已冻结", settled:"已结算", released:"已解冻" };

export default function AdminGenerationTasksPage() {
    const token = useUserStore((state) => state.token);
	const actionRef = useRef<ActionType>(null);
	const [form] = Form.useForm();
	const [importOpen,setImportOpen] = useState(false);
	const [importing,setImporting] = useState(false);
	const [preview,setPreview] = useState<AdminGenerationTask|null>(null);
    const columns: ProColumns<AdminGenerationTask>[] = [
        {title:"搜索",dataIndex:"keyword",hideInTable:true},
        {title:"用户",dataIndex:"userDisplayName",width:160,search:false,render:(_,item)=><Space direction="vertical" size={0}><Typography.Text>{item.userDisplayName||item.userId}</Typography.Text><Typography.Text type="secondary" copyable>{item.userId}</Typography.Text></Space>},
        {title:"类型",dataIndex:"kind",width:80,valueType:"select",valueEnum:{image:{text:"图片"},video:{text:"视频"},audio:{text:"音频"}},render:(_,item)=><Tag>{kindLabels[item.kind]}</Tag>},
        {title:"模型档位",dataIndex:"model",width:220,ellipsis:true,search:false},
		{title:"中转站",dataIndex:"channelName",width:130,ellipsis:true,search:false,render:(_,item)=>item.channelName||"-"},
		{title:"上游任务",dataIndex:"upstreamTaskId",width:150,ellipsis:true,search:false,render:(_,item)=>item.upstreamTaskId?<Typography.Text copyable>{item.upstreamTaskId}</Typography.Text>:"-"},
        {title:"任务状态",dataIndex:"status",width:130,valueType:"select",valueEnum:{queued:{text:"排队"},processing:{text:"处理中"},reconciling:{text:"等待上游确认"},completed:{text:"成功"},failed:{text:"失败"}},render:(_,item)=><Tag color={item.status==="completed"?"success":item.status==="failed"?"error":item.status==="reconciling"?"warning":"processing"}>{item.status==="reconciling"?"等待上游确认":item.status}</Tag>},
        {title:"资金状态",dataIndex:"billingStatus",width:110,valueType:"select",valueEnum:{frozen:{text:"已冻结"},settled:{text:"已结算"},released:{text:"已解冻"}},render:(_,item)=><Tag color={item.billingStatus==="settled"?"success":item.billingStatus==="released"?"default":"warning"}>{billingLabels[item.billingStatus]||"未计费"}</Tag>},
        {title:"售价",dataIndex:"priceCents",width:100,render:(_,item)=>formatCNY(item.priceCents)},
		{title:"生成效果",dataIndex:"resultUrl",width:190,search:false,fixed:"right",render:(_,item)=>!item.resultUrl?<Typography.Text type="secondary">暂无结果</Typography.Text>:<Space direction="vertical" size={4}>{item.kind==="image"?<Image src={item.resultUrl} width={120} height={76} preview={false} style={{objectFit:"cover",borderRadius:8}}/>:item.kind==="video"?<video src={item.resultUrl} muted preload="metadata" style={{width:150,height:84,objectFit:"cover",borderRadius:8,background:"#000"}}/>:<audio src={item.resultUrl} preload="metadata" style={{width:170}}/>}<Button size="small" type="primary" onClick={()=>setPreview(item)}>放大预览</Button></Space>},
        {title:"错误",dataIndex:"error",ellipsis:true,render:(_,item)=><Typography.Text type={item.error?"danger":"secondary"}>{item.error||"-"}</Typography.Text>},
        {title:"创建时间",dataIndex:"createdAt",width:180,search:false,render:(_,item)=>item.createdAt?dayjs(item.createdAt).format("YYYY-MM-DD HH:mm:ss"):"-"},
        {title:"创建日期",dataIndex:"createdRange",valueType:"dateRange",hideInTable:true},
    ];
    return <main style={{padding:24}}><Card variant="borderless"><ProTable actionRef={actionRef} rowKey="id" columns={columns} scroll={{x:1500}} request={async(params)=>{ if(!token)return{data:[],success:true,total:0}; const range=params.createdRange as string[]|undefined; const data=await fetchAdminGenerationTasks(token,{keyword:params.keyword as string,kind:params.kind as string,status:params.status as string,billingStatus:params.billingStatus as string,startedAt:range?.[0],endedAt:range?.[1],limit:500}); return{data,success:true,total:data.length}; }} pagination={{pageSize:20}} headerTitle={<Typography.Text strong>生成任务</Typography.Text>} toolBarRender={()=>[<Button key="import" onClick={()=>setImportOpen(true)}>导入上游结果</Button>,<Button key="refresh" icon={<ReloadOutlined/>} onClick={()=>actionRef.current?.reload()}>刷新</Button>]} /></Card><Modal title="导入上游生成结果" open={importOpen} confirmLoading={importing} onCancel={()=>setImportOpen(false)} onOk={async()=>{try{const values=await form.validateFields();setImporting(true);await importAdminGenerationTask(token,values);message.success("已导入");setImportOpen(false);form.resetFields();actionRef.current?.reload();}finally{setImporting(false);}}}><Form form={form} layout="vertical" initialValues={{status:"completed",channelName:"WaveSpeed"}}><Form.Item name="userId" label="用户 ID" rules={[{required:true}]}><Input/></Form.Item><Form.Item name="userDisplayName" label="用户名"><Input/></Form.Item><Form.Item name="model" label="模型" rules={[{required:true}]}><Input/></Form.Item><Form.Item name="channelName" label="中转站" rules={[{required:true}]}><Input/></Form.Item><Form.Item name="upstreamTaskId" label="上游任务 ID" rules={[{required:true}]}><Input/></Form.Item><Form.Item name="status" label="状态"><Select options={[{label:"成功",value:"completed"},{label:"失败",value:"failed"}]}/></Form.Item><Form.Item name="resultUrl" label="结果地址"><Input/></Form.Item><Form.Item name="error" label="失败原因"><Input/></Form.Item><Form.Item name="createdAt" label="创建时间（ISO 或 RFC3339）"><Input placeholder="2026-08-29T21:00:43+08:00"/></Form.Item></Form></Modal><Modal title={`${preview?.kind==="video"?"视频":preview?.kind==="image"?"图片":"音频"}生成效果`} open={Boolean(preview)} footer={null} width={900} onCancel={()=>setPreview(null)} destroyOnHidden>{preview?.kind==="video"?<video src={preview.resultUrl} controls autoPlay style={{width:"100%",maxHeight:"70vh",background:"#000"}}/>:preview?.kind==="image"?<Image src={preview.resultUrl} width="100%"/>:preview?<audio src={preview.resultUrl} controls autoPlay style={{width:"100%"}}/>:null}</Modal></main>;
}
