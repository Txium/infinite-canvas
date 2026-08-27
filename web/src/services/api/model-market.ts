import { apiGet } from "@/services/api/request";

export type MarketModelCard = {
    id: string; name: string; category: "llm" | "image" | "video" | "person" | "music" | "voice" | "3d" | "tool";
    icon: string; description: string; modes: string[]; resolutions: string[]; durations: string[]; ratios: string[];
    maxReferenceImages: number; supportsPerson: boolean; supportsFirstLastFrame: boolean; supportsAudioReference: boolean;
    speed: string; featured: boolean; status: "normal" | "busy" | "maintenance"; available: boolean; availableVariantIds: string[];
    variants: MarketModelVariant[];
};

export type MarketModelVariant = {
    id: string; modelId: string; name: string; priceCents?: number; priceText: string; billingUnit: string;
    pricingMode: "fixed" | "dynamic" | "disabled"; priceFormula: string;
    personNote: string; remark: string; enabled: boolean; sort: number;
};

export async function fetchModelMarket() {
    const items = await apiGet<MarketModelCard[]>("/api/model-market");
    return items.map((item) => ({
        ...item,
        modes: item.modes || [],
        resolutions: item.resolutions || [],
        durations: item.durations || [],
        ratios: item.ratios || [],
        variants: item.variants || [],
        availableVariantIds: item.availableVariantIds || [],
    }));
}
