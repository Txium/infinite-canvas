"use client";

import { Badge, Button, Card, Progress, Space, Typography, theme } from "antd";
import Link from "next/link";
import { useEffect, useState } from "react";

import { fetchUserGenerationTasks, type UserGenerationTask } from "@/services/api/generation-tasks";
import { useUserStore } from "@/stores/use-user-store";

export function RecentGenerationTasksPanel() {
    const token=useUserStore((state)=>state.token); const [items,setItems]=useState<UserGenerationTask[]>([]); const {token:themeToken}=theme.useToken();
    useEffect(()=>{if(!token)return; const load=()=>fetchUserGenerationTasks(token,{limit:3}).then(setItems).catch(()=>{}); void load(); const timer=window.setInterval(load,5000); return()=>window.clearInterval(timer)},[token]);
    if(!token||!items.length)return null;
    return <Card size="small" title="最近任务" extra={<Link href="/generation-history">全部</Link>} className="pointer-events-auto fixed bottom-4 right-4 z-30 w-72" style={{background:themeToken.colorBgContainer,borderColor:themeToken.colorBorder}}>{items.map((item)=><div key={item.task_id} className="mb-2 last:mb-0"><Space className="w-full justify-between"><Typography.Text ellipsis className="max-w-44">{item.display_model_name}</Typography.Text><Badge status={item.status==="completed"?"success":item.status==="failed"||item.status==="refunded"?"error":"processing"} text={item.status==="completed"?"成功":item.status==="refunded"?"已退款":item.status==="failed"?"失败":"生成中"}/></Space>{item.status==="processing"||item.status==="queued"?<Progress percent={item.progress||0} size="small" showInfo={false}/>:null}</div>)}</Card>;
}
