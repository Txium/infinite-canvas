"use client";

import { useEffect, useRef, useState } from "react";
import { Button, Checkbox, InputNumber, Modal } from "antd";
import { loadProcessingSource, processOriginalMedia, type MediaAction, type MediaProbe } from "@/services/api/media-worker";
import type { CanvasNodeData } from "../types";

export function CanvasMediaWorkbench({node, onClose, onOutput}:{node:CanvasNodeData; onClose:()=>void; onOutput:(blob:Blob, action:MediaAction, time:number, continueVideo:boolean|"image", metadata:MediaProbe)=>Promise<void>}) {
    const player = useRef<HTMLVideoElement>(null);
    const [file,setFile] = useState<Blob>();
    const [url,setUrl] = useState("");
    const [thumbnailUrl,setThumbnailUrl] = useState("");
    useEffect(()=>()=>{if(thumbnailUrl)URL.revokeObjectURL(thumbnailUrl);},[thumbnailUrl]);
    const [meta,setMeta] = useState<MediaProbe>();
    const [time,setTime] = useState(0);
    const [start,setStart] = useState(0);
    const [end,setEnd] = useState(0);
    const [precise,setPrecise] = useState(false);
    const [busy,setBusy] = useState(true);
    const [error,setError] = useState("");
    useEffect(()=>{
        let disposed=false, objectUrl="";
        void (async()=>{
            try {
                const original=await loadProcessingSource(node.metadata?.storageKey,node.metadata?.content);
                const info=await processOriginalMedia(original,"probe").then(r=>r.json()) as MediaProbe;
                if(disposed) return;
                objectUrl=URL.createObjectURL(original);
                setFile(original); setUrl(objectUrl); setMeta(info); setEnd(info.duration);
            } catch(e) {if(!disposed) setError(e instanceof Error?e.message:"读取原视频失败");}
            finally {if(!disposed) setBusy(false);}
        })();
        return ()=>{disposed=true; if(objectUrl) URL.revokeObjectURL(objectUrl);};
    },[node.id,node.metadata?.content,node.metadata?.storageKey]);
    const frameIndex = meta ? Math.max(0, meta.frameTimes.findLastIndex(t=>t<=time+.0001)) : 0;
    const seek = (value:number)=>{if(player.current && meta){player.current.pause(); player.current.currentTime=Math.max(0,Math.min(value,meta.duration-.001)); setTime(player.current.currentTime);}};
    const stepFrame=(offset:number)=>{if(meta) seek(meta.frameTimes[Math.max(0,Math.min(frameIndex+offset,meta.frameTimes.length-1))] ?? time+offset/meta.fps);};
    async function run(action:MediaAction, continueVideo:boolean|"image"=false) {
        if(!file || !meta) return;
        setBusy(true);setError("");
        try {const blob=await processOriginalMedia(file,action,action==="clip"?start:time,end,precise).then(r=>r.blob()); if(action==="thumbnails")setThumbnailUrl(URL.createObjectURL(blob));else await onOutput(blob,action,time,continueVideo,meta);}
        catch(e){setError(e instanceof Error?e.message:"处理失败");}
        finally{setBusy(false);}
    }
    const seconds=Math.floor(time), fps=Math.round(meta?.fps||25);
    const timecode=[Math.floor(seconds/3600),Math.floor(seconds/60)%60,seconds%60,Math.min(fps-1,Math.floor((time-seconds)*fps))].map(v=>String(v).padStart(2,"0")).join(":");
    return <Modal open title="视频处理 · 原文件" width={880} footer={null} onCancel={busy?undefined:onClose} closable={!busy} maskClosable={!busy}>
        <div className="space-y-3">
            {error && <p role="alert" className="text-red-500 whitespace-pre-wrap">{error}</p>}
            {busy && <p role="status">正在处理，请勿重复提交；不消耗模型余额。</p>}
            {url && <video ref={player} src={url} controls className="w-full max-h-[45vh]" onTimeUpdate={e=>setTime(e.currentTarget.currentTime)}/>}
            {meta && <>
                {!meta.audioCodec && <p role="status">VIDEO_HAS_NO_AUDIO：原视频没有音频轨，不能提取音频。仍可截帧或裁剪。</p>}
                <Button disabled={busy} onClick={()=>void run("thumbnails")}>加载低清时间轴缩略图</Button>
                {thumbnailUrl && <div className="grid grid-cols-8">{Array.from({length:8},(_,i)=><button key={i} title={`${(meta.duration*i/8).toFixed(2)}秒`} aria-label={`跳转缩略图${i+1}`} onClick={()=>seek(meta.duration*i/8)} style={{aspectRatio:"16 / 9",backgroundImage:`url(${thumbnailUrl})`,backgroundSize:"800% 100%",backgroundPosition:`${i/7*100}% 0`}}/>)}</div>}
                <input aria-label="视频处理时间轴" type="range" className="w-full" min={0} max={meta.duration} step={.001} value={time} onChange={e=>seek(Number(e.target.value))}/>
                <div className="font-mono">{timecode} · 第 {frameIndex+1} 帧 · {meta.fps.toFixed(3)} fps · {meta.width}×{meta.height} · {meta.codec} · {meta.duration.toFixed(2)}秒 · {meta.bitrate?`${Math.round(Number(meta.bitrate)/1000)} kbps`:"码率未知"}</div>
                <div className="flex flex-wrap gap-2">
                    <Button onClick={()=>seek(time-5)}>−5秒</Button><Button onClick={()=>seek(time-1)}>−1秒</Button>
                    <Button onClick={()=>stepFrame(-1)}>上一帧</Button><Button onClick={()=>{if(player.current?.paused) void player.current.play(); else player.current?.pause();}}>播放/暂停</Button><Button onClick={()=>stepFrame(1)}>下一帧</Button>
                    <Button onClick={()=>seek(time+1)}>+1秒</Button><Button onClick={()=>seek(time+5)}>+5秒</Button>
                </div>
                <div className="flex flex-wrap gap-2">
                    <Button disabled={busy} onClick={()=>void run("first")}>截取首帧</Button><Button disabled={busy} onClick={()=>void run("last")}>截取尾帧</Button>
                    <Button disabled={busy} onClick={()=>void run("frame")}>截取当前帧</Button><Button disabled={busy} onClick={()=>void run("last",true)}>尾帧接下一镜</Button>
                    <Button disabled={busy} onClick={()=>void run("frame",true)}>当前帧作为下一镜参考</Button>
                    <Button disabled={busy} onClick={()=>void run("frame","image")}>当前帧作为生图参考</Button>
                    <Button disabled={busy||!meta.audioCodec} onClick={()=>void run("audio")}>提取音频</Button><Button disabled={busy} onClick={()=>void run("mute")}>提取无声视频</Button><Button disabled={busy||!meta.subtitleTracks} onClick={()=>void run("subtitles")}>移除独立字幕轨</Button>
                </div>
                <div className="flex flex-wrap items-center gap-2">
                    起点 <InputNumber aria-label="片段起点" min={0} max={meta.duration} value={start} onChange={v=>setStart(v||0)}/>
                    终点 <InputNumber aria-label="片段终点" min={0} max={meta.duration} value={end} onChange={v=>setEnd(v||0)}/>
                    <Checkbox checked={precise} onChange={e=>setPrecise(e.target.checked)}>精确裁剪（重新编码）</Checkbox>
                    <Button disabled={busy||end<=start} onClick={()=>void run("clip")}>生成片段节点</Button>
                </div>
                <p className="text-xs opacity-70">默认裁剪为流复制，边界受关键帧约束；精确裁剪会重新编码。截图从原视频解码，不截播放器画面。移除字幕轨不能去掉烧录字幕。原视频始终保留。</p>
            </>}
        </div>
    </Modal>;
}
