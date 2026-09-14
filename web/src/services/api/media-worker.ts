import { useUserStore } from "@/stores/use-user-store";
import { downloadRemoteMedia, resolveMediaUrl } from "@/services/file-storage";

export type MediaProbe = { duration: number; fps: number; frameTimes: number[]; width: number; height: number; codec: string; bitrate?: string; bytes: number; audioCodec?: string; subtitleTracks: number };
export type MediaAction = "probe" | "first" | "last" | "frame" | "clip" | "audio" | "mute" | "subtitles" | "thumbnails";

export async function loadProcessingSource(storageKey?: string, content = "") {
    const token = useUserStore.getState().token;
    if (!token) throw new Error("请先登录后使用视频处理");
    const state = await fetch("/api/v1/media-worker", {headers:{Authorization:`Bearer ${token}`}}).then(r=>r.json());
    if (!state.data?.configured) throw new Error("媒体Worker尚未部署。请管理员配置 MEDIA_WORKER_URL 和 MEDIA_WORKER_TOKEN；此功能不调用付费模型。");
    const maxBytes = Number(state.data.maxBytes);
    if (!Number.isSafeInteger(maxBytes) || maxBytes < 1) throw new Error("媒体Worker上传限制配置无效");
    const url = await resolveMediaUrl(storageKey, content);
    const blob = url.startsWith("blob:") || url.startsWith("data:") ? await fetch(url).then(r=>r.blob()) : await downloadRemoteMedia(url);
    if (blob.size > maxBytes) throw new Error(`当前媒体处理支持${Math.floor(maxBytes/1024/1024)}MB以内的原视频，请先选择短片段`);
    return blob;
}

export async function processOriginalMedia(file: Blob, action: MediaAction, start=0, end=0, precise=false) {
    const body = new FormData();
    body.append("file", file, "source-video");
    body.append("action", action);
    body.append("start", String(start));
    body.append("end", String(end));
    body.append("precise", String(precise));
    const response = await fetch("/api/v1/media-worker/process", {method:"POST", headers:{Authorization:`Bearer ${useUserStore.getState().token}`}, body, signal:AbortSignal.timeout(155000)});
    if (!response.ok) {
        const error = await response.json().catch(()=>({msg:"媒体处理连接失败"}));
        throw new Error(`${error.error_code ? `[${error.error_code}] ` : ""}${error.msg || "媒体处理失败"}`);
    }
    return response;
}
