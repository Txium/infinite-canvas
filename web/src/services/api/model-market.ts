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
    return items.flatMap((item) => {
        const availableVariantIds = item.availableVariantIds || [];
        const available = new Set(availableVariantIds);
        // Keep the browser resilient to an older/cached API response: a
        // variant without an enabled route must never be rendered as a
        // selectable “线路配置中” option in the public market.
        const variants = (item.variants || []).filter((variant) => available.has(variant.id));
        if (!variants.length) return [];
        return [{
            ...item,
            modes: item.modes || [],
            resolutions: item.resolutions || [],
            durations: item.durations || [],
            ratios: item.ratios || [],
            variants,
            availableVariantIds: availableVariantIds.filter((id) => variants.some((variant) => variant.id === id)),
            available: true,
        }];
    });
}
