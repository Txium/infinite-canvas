import { useUserStore } from "@/stores/use-user-store";
import { downloadRemoteMedia, resolveMediaUrl } from "@/services/file-storage";

export type MediaProbe = { duration: number; fps: number; frameTimes: number[]; width: number; height: number; codec: string; bitrate?: string; bytes: number; audioCodec?: string; subtitleTracks: number };
export type MediaAction = "probe" | "first" | "last" | "frame" | "clip" | "audio" | "mute" | "subtitles" | "thumbnails";

type MediaWorkerState = { configured?: boolean; ready?: boolean; maxBytes?: number; msg?: string };

async function readErrorResponse(response: Response, fallback: string) {
    const text = await response.text().catch(() => "");
    if (text) {
        try {
            const payload = JSON.parse(text) as { error_code?: string; msg?: string; message?: string };
            const message = payload.msg || payload.message;
            if (message) return `${payload.error_code ? `[${payload.error_code}] ` : ""}${message}`;
        } catch {
            if (!/^\s*</.test(text)) return text.slice(0, 300);
        }
    }
    return `${fallback}（HTTP ${response.status}）`;
}

export async function loadProcessingSource(storageKey?: string, content = "") {
    const token = useUserStore.getState().token;
    if (!token) throw new Error("请先登录后使用视频处理");
    const response = await fetch("/api/v1/media-worker", {headers:{Authorization:`Bearer ${token}`}});
    if (!response.ok) throw new Error(await readErrorResponse(response, "无法检查媒体 Worker 状态"));
    const state = await response.json() as { data?: MediaWorkerState };
    if (!state.data?.configured) throw new Error("媒体 Worker 尚未配置；此功能不调用付费模型。");
    // The readiness probe also wakes a sleeping free Render worker. It is
    // advisory: a transient health response must not block the real request,
    // whose status and Worker error body are authoritative.
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
    const response = await fetch("/api/v1/media-worker/process", {method:"POST", headers:{Authorization:`Bearer ${useUserStore.getState().token}`}, body, signal:AbortSignal.timeout(245000)});
    if (!response.ok) {
        throw new Error(await readErrorResponse(response, "媒体处理请求失败"));
    }
    return response;
}
