import { apiGet } from "@/services/api/request";

export type MarketModelCard = {
    id: string; name: string; category: "image" | "video" | "person" | "audio" | "tool";
    icon: string; description: string; modes: string[]; resolutions: string[]; durations: string[]; ratios: string[];
    maxReferenceImages: number; supportsPerson: boolean; supportsFirstLastFrame: boolean; supportsAudioReference: boolean;
    speed: string; featured: boolean; status: "normal" | "busy" | "maintenance"; available: boolean;
    prices: Array<{ variant: string; billingMode: string; unit: string; priceCredits: number; currency: string }>;
};

export function fetchModelMarket() { return apiGet<MarketModelCard[]>("/api/model-market"); }
