"use client";

import { App, Button, Card, Empty, Segmented, Space, Tag, Typography } from "antd";
import dayjs from "dayjs";
import { Copy, Download, RefreshCw, RotateCcw } from "lucide-react";
import Link from "next/link";
import { useEffect, useState } from "react";

import { formatCNY } from "@/constant/credits";
import { fetchUserGenerationTasks, type UserGenerationTask } from "@/services/api/generation-tasks";
import { useUserStore } from "@/stores/use-user-store";

const statusText: Record<string,string> = { queued:"排队中",processing:"生成中",completed:"成功",failed:"失败",refunded:"已退款" };

export default function GenerationHistoryPage() {
    const { message } = App.useApp();
    const token=useUserStore((state)=>state.token); const user=useUserStore((state)=>state.user);
    const [items,setItems]=useState<UserGenerationTask[]>([]); const [kind,setKind]=useState("all"); const [status,setStatus]=useState("all"); const [days,setDays]=useState(0);
    const load=async()=>{if(token)setItems(await fetchUserGenerationTasks(token,{kind:kind==="all"?undefined:kind,status:status==="all"?undefined:status,days:days||undefined,limit:200}))};
    useEffect(()=>{void load(); if(!token)return; const timer=window.setInterval(()=>void load(),5000); return()=>window.clearInterval(timer)},[token,kind,status,days]);
    if(!user)return <main className="grid h-full place-items-center"><Typography.Text>请先登录后查看生成记录</Typography.Text></main>;
    return <main className="h-full overflow-y-auto p-6"><div className="mx-auto max-w-6xl space-y-4"><Space className="w-full justify-between"><div><Typography.Title level={2} className="!mb-1">生成记录</Typography.Title><Typography.Text type="secondary">只展示你自己的任务、消费、退款和结果。</Typography.Text></div><Button icon={<RefreshCw className="size-4"/>} onClick={()=>void load()}>刷新</Button></Space><Space wrap><Segmented value={kind} onChange={(v)=>setKind(String(v))} options={[{label:"全部",value:"all"},{label:"图片",value:"image"},{label:"视频",value:"video"},{label:"音频",value:"audio"}]}/><Segmented value={status} onChange={(v)=>setStatus(String(v))} options={[{label:"全部状态",value:"all"},{label:"成功",value:"completed"},{label:"失败",value:"failed"},{label:"已退款",value:"refunded"}]}/><Segmented value={days} onChange={(v)=>setDays(Number(v))} options={[{label:"累计",value:0},{label:"最近7天",value:7},{label:"最近30天",value:30}]}/></Space>{items.length?<div className="grid gap-4 md:grid-cols-2">{items.map((item)=><Card key={`${item.task_type}-${item.task_id}`} title={`${item.display_model_name}${item.variant_name&&item.variant_name!==item.display_model_name?` · ${item.variant_name}`:""}`} extra={<Tag color={item.status==="completed"?"green":item.status==="refunded"?"blue":item.status==="failed"?"red":"gold"}>{statusText[item.status]||item.status}</Tag>}><div className="grid grid-cols-[140px_1fr] gap-y-2 text-sm"><span className="text-stone-500">任务类型</span><span>{item.task_type==="video"?"视频":item.task_type==="image"?"图片":"音频"}</span><span className="text-stone-500">任务时间</span><span>{dayjs(item.created_at).format("YYYY-MM-DD HH:mm:ss")}</span><span className="text-stone-500">消费金额</span><span>{formatCNY(item.sale_price_cents)}</span><span className="text-stone-500">生成耗时</span><span>{item.duration_seconds?`${item.duration_seconds} 秒`:"-"}</span>{item.refund_amount_cents>0?<><span className="text-stone-500">退款</span><span className="text-green-600">已退回 {formatCNY(item.refund_amount_cents)}</span></>:null}</div>{item.input_summary?<Typography.Paragraph ellipsis={{rows:2}} className="!mt-3 !mb-0">{item.input_summary}</Typography.Paragraph>:null}{item.user_friendly_error?<Typography.Text type="danger">{item.user_friendly_error}</Typography.Text>:null}{item.result_url?<div className="mt-3">{item.task_type==="image"?<img src={item.result_url} alt="生成结果" className="max-h-56 rounded-lg object-contain"/>:item.task_type==="video"?<video src={item.result_url} controls className="max-h-56 w-full rounded-lg"/>:<audio src={item.result_url} controls className="w-full"/>}</div>:null}<Space wrap className="mt-3"><Button icon={<Download className="size-4"/>} href={item.result_url||undefined} target={item.result_url?"_blank":undefined} disabled={!item.result_url}>查看 / 下载</Button><Button icon={<Copy className="size-4"/>} disabled={!item.input_summary} onClick={async()=>{await navigator.clipboard.writeText(item.input_summary);message.success("参数已复制")}}>复制参数</Button><Link href="/canvas"><Button icon={<RotateCcw className="size-4"/>}>去画布重新生成</Button></Link></Space></Card>)}</div>:<Empty description="暂无生成记录"/>}</div></main>;
}
