import { apiGet, compactApiParams } from "@/services/api/request";
import { starterPrompts } from "@/constant/starter-library";

export type Prompt = {
    id: string;
    title: string;
    coverUrl: string;
    prompt: string;
    tags: string[];
    category: string;
    githubUrl: string;
    preview: string;
    createdAt: string;
    updatedAt: string;
};

export const ALL_PROMPTS_OPTION = "全部";

export type PromptListResponse = {
    items: Prompt[];
    tags: string[];
    categories: string[];
    total: number;
};

export async function fetchPrompts({ keyword = "", tag = [], category = ALL_PROMPTS_OPTION, page, pageSize }: { keyword?: string; tag?: string[]; category?: string; page?: number; pageSize?: number } = {}) {
    const params = compactApiParams({ ...(keyword ? { keyword } : {}), ...(tag.length ? { tag } : {}), ...(category !== ALL_PROMPTS_OPTION ? { category } : {}), ...(page ? { page } : {}), ...(pageSize ? { pageSize } : {}) });
    try {
        const remote = await apiGet<PromptListResponse>("/api/prompts", params);
        if (remote.total > 0) return remote;
    } catch {}
    const keywordValue = keyword.trim().toLowerCase();
    const filtered = starterPrompts.filter((item) => (!keywordValue || `${item.title} ${item.prompt} ${item.tags.join(" ")}`.toLowerCase().includes(keywordValue)) && (category === ALL_PROMPTS_OPTION || item.category === category) && (!tag.length || tag.every((value) => item.tags.includes(value))));
    const size = pageSize || 20;
    const start = ((page || 1) - 1) * size;
    return { items: filtered.slice(start, start + size), tags: unique(starterPrompts.flatMap((item) => item.tags)), categories: unique(starterPrompts.map((item) => item.category)), total: filtered.length };
}

function unique(values: string[]) {
    return Array.from(new Set(values));
}

export function formatPromptDate(value: string) {
    const date = new Date(value);
    return Number.isNaN(date.getTime()) ? "" : new Intl.DateTimeFormat("zh-CN", { year: "numeric", month: "2-digit", day: "2-digit" }).format(date);
}

