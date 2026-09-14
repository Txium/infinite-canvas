import { useUserStore } from "@/stores/use-user-store";

export type MediaCapability = {operation:string;name:string;status:"COMING_SOON"|"DISABLED"};
export type VoiceProfile = {character_id:string;character_name:string;voice_provider:string;voice_id:string;voice_prompt:string;default_speed:number;default_emotion:string;default_style:string};

async function request<T>(path:string, body?:unknown):Promise<T> {
 const token=useUserStore.getState().token;
 if(!token) throw new Error("请先登录");
 const response=await fetch(`/api/v1/${path}`,{method:body?"POST":"GET",headers:{Authorization:`Bearer ${token}`,"Content-Type":"application/json"},...(body?{body:JSON.stringify(body)}:{}),signal:AbortSignal.timeout(15000)});
 const data=await response.json();
 if(!response.ok || data.code!==0) throw new Error(data.msg||"请求失败");
 return data.data;
}
export const getMediaCapabilities=()=>request<MediaCapability[]>("media-ai/capabilities");
export const getVoiceProfiles=()=>request<VoiceProfile[]>("voice-profiles");
export const saveVoiceProfile=(value:VoiceProfile)=>request<VoiceProfile>("voice-profiles",value);
