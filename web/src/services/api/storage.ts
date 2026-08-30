import { apiDelete, apiGet, apiPost } from "@/services/api/request";
import { useUserStore } from "@/stores/use-user-store";
import type { UserWebDAVStorageProvider } from "@/services/image-storage";

export type RegisteredStorageObject = {
    url: string;
    storageKey: string;
    bytes: number;
    mimeType: string;
};

export type StorageObjectInfo = {
    id: string;
    objectKey: string;
    publicUrl: string;
    mimeType: string;
    bytes: number;
    direct: boolean;
};

export function getStorageObjectInfo(id: string) {
    const token = useUserStore.getState().token;
    if (!token) return Promise.reject(new Error("请先登录后读取文件信息"));
    return apiGet<StorageObjectInfo>(`/api/v1/files/${encodeURIComponent(id)}`, undefined, token);
}

export function registerDirectStorageObject(
    token: string,
    payload: { provider: UserWebDAVStorageProvider; objectKey: string; mimeType: string; bytes: number },
) {
    return apiPost<RegisteredStorageObject>("/api/v1/files/direct", payload, token);
}

export function deleteDirectStorageObjectRecord(token: string, id: string) {
    return apiDelete<boolean>(`/api/v1/files/${encodeURIComponent(id)}/record`, token);
}
