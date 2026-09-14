"use client";

import { useEffect, useState } from "react";
import { Modal } from "antd";
import { getMediaCapabilities, getVoiceProfiles, saveVoiceProfile, type MediaCapability, type VoiceProfile } from "@/services/api/media-ai";

export function CanvasAudioAIPanel({characterId,characterName,onClose}:{characterId:string;characterName:string;onClose:()=>void}) {
 const [capabilities,setCapabilities]=useState<MediaCapability[]>([]);
 const [profiles,setProfiles]=useState<VoiceProfile[]>([]);
 const [profile,setProfile]=useState<VoiceProfile>({character_id:characterId,character_name:characterName,voice_provider:"",voice_id:"",voice_prompt:"",default_speed:1,default_emotion:"",default_style:""});
 const [status,setStatus]=useState("");
 const [busy,setBusy]=useState(true);
 useEffect(()=>{let active=true;Promise.all([getMediaCapabilities(),getVoiceProfiles()]).then(([c,p])=>{if(active){setCapabilities(c);setProfiles(p);const saved=p.find(v=>v.character_id===characterId);if(saved)setProfile(saved);}}).catch(e=>{if(active)setStatus(e.message);}).finally(()=>{if(active)setBusy(false);});return()=>{active=false};},[characterId]);
 async function save(){setBusy(true);try{const saved=await saveVoiceProfile(profile);setProfiles(p=>[saved,...p.filter(v=>v.character_id!==saved.character_id)]);setStatus("音色档案已保存；保存不验证Voice ID，不调用模型。");}catch(e){setStatus(e instanceof Error?e.message:"保存失败");}finally{setBusy(false);}}
 return <Modal open title="角色音色与媒体AI" footer={null} onCancel={onClose} width={720}>
  <p>当前为接口预留阶段。以下能力尚未接入执行模型，不会提交生成或扣费。</p>
  <div className="my-3 flex flex-wrap gap-2">{capabilities.map(c=><button key={c.operation} disabled title="未配置执行Adapter" className="rounded border border-current px-2 py-1 opacity-60">{c.name} · 待接入</button>)}</div>
  <p className="text-xs opacity-70">说话人识别输出谁在什么时间讲话；重叠语音分离输出独立音轨，两者不同。字幕轨删除在视频处理里，AI去画面字幕另需模型。</p>
  <h3 className="mt-4 mb-2">角色 Voice Profile</h3>
  <p className="mb-2 text-xs opacity-70">默认绑定打开面板的节点ID；也可填写你统一使用的角色ID。这里只保存音色档案，不生成试听或克隆声音。不要填写API Key。</p>
  <label className="block mb-2">已有角色<select className="ml-2 border border-current rounded bg-transparent" value="" onChange={e=>{const p=profiles.find(v=>v.character_id===e.target.value);if(p)setProfile(p);}}><option value="">选择已保存档案</option>{profiles.map(p=><option key={p.character_id} value={p.character_id}>{p.character_name||p.character_id}</option>)}</select></label>
  <div className="grid grid-cols-2 gap-3">{([{key:"character_id",label:"角色ID"},{key:"character_name",label:"角色名称"},{key:"voice_provider",label:"音色Provider标识"},{key:"voice_id",label:"Voice ID"},{key:"default_emotion",label:"默认情绪"},{key:"default_style",label:"默认风格"}] as const).map(f=><label key={f.key}>{f.label}<input className="block w-full rounded border border-current bg-transparent p-2" value={profile[f.key]} maxLength={f.key==="character_id"?128:200} onChange={e=>setProfile(p=>({...p,[f.key]:e.target.value}))}/></label>)}</div>
  <label className="block mt-3">音色描述<textarea className="block w-full border border-current rounded bg-transparent p-2" rows={3} maxLength={4000} value={profile.voice_prompt} onChange={e=>setProfile(p=>({...p,voice_prompt:e.target.value}))}/></label>
  <label className="block my-3">默认语速 <input type="number" min={.25} max={4} step={.05} className="border border-current rounded bg-transparent p-1" value={profile.default_speed} onChange={e=>setProfile(p=>({...p,default_speed:Number(e.target.value)}))}/></label>
  <button disabled={busy||!profile.character_id.trim()} className="border border-current rounded px-3 py-2 disabled:opacity-50" onClick={()=>void save()}>保存角色音色</button>
  {status&&<p role="status" className="mt-2">{status}</p>}
 </Modal>;
}
