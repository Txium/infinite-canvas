import { apiGet } from "@/services/api/request";

export type MarketModelCard = {
    id: string; name: string; category: "llm" | "image" | "video" | "person" | "music" | "voice" | "3d" | "tool";
    icon: string; description: string; modes: string[]; resolutions: string[]; durations: string[]; ratios: string[];
    maxReferenceImages: number; supportsPerson: boolean; supportsFirstLastFrame: boolean; supportsAudioReference: boolean;
    speed: string; featured: boolean; status: "normal" | "busy" | "maintenance"; available: boolean;
    prices: Array<{ variant: string; billingMode: string; unit: string; priceCredits: number; currency: string }>;
};

export async function fetchModelMarket() {
    const items = await apiGet<MarketModelCard[]>("/api/model-market");
    return items.map((item) => ({
        ...item,
        modes: item.modes || [],
        resolutions: item.resolutions || [],
        durations: item.durations || [],
        ratios: item.ratios || [],
        prices: item.prices || [],
    }));
}
