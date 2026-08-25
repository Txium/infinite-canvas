import { apiGet, compactApiParams } from "@/services/api/request";
import { starterAssets } from "@/constant/starter-library";

export type AssetLibraryItem = {
    id: string;
    title: string;
    type: "text" | "image" | "video" | "audio";
    coverUrl: string;
    tags: string[];
    category: string;
    description: string;
    content: string;
    url: string;
    createdAt: string;
    updatedAt: string;
};

export type AssetLibraryResponse = {
    items: AssetLibraryItem[];
    tags: string[];
    total: number;
};

export type AssetLibraryQuery = {
    keyword?: string;
    type?: string;
    tag?: string[];
    page?: number;
    pageSize?: number;
};

export async function fetchAssetLibrary(query: AssetLibraryQuery = {}) {
    try {
        const remote = await apiGet<AssetLibraryResponse>("/api/assets", compactApiParams(query));
        if (remote.total > 0) return remote;
    } catch {}
    const keyword = query.keyword?.trim().toLowerCase() || "";
    const tags = query.tag || [];
    const filtered = starterAssets.filter((item) => (!keyword || `${item.title} ${item.content} ${item.tags.join(" ")}`.toLowerCase().includes(keyword)) && (!query.type || item.type === query.type) && (!tags.length || tags.every((tag) => item.tags.includes(tag))));
    const size = query.pageSize || 12;
    const start = ((query.page || 1) - 1) * size;
    return { items: filtered.slice(start, start + size), tags: Array.from(new Set(starterAssets.flatMap((item) => item.tags))), total: filtered.length };
}

