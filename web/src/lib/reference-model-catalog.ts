export type ReferenceCatalogCapability = "image" | "video" | "text";

export type ReferenceCatalogModel = {
    model: string;
    capability: ReferenceCatalogCapability;
    source: string;
};

// Model identifiers adapted for personal use from hero8152/Infinite-Canvas.
// The source repository is credited here as required by its license. These are
// catalog entries only: requests always use the current user's own provider Key.
const groups: Array<{ source: string; capability: ReferenceCatalogCapability; models: string[] }> = [
    { source: "通用语言模型", capability: "text", models: ["gpt-5.5", "gpt-4o-mini", "openai/gpt-5.1", "google/gemini-3.1-flash-lite-preview", "qwen/qwen3-vl-235b-a22b-instruct", "qwen/qwen-plus"] },
    { source: "ModelScope", capability: "text", models: ["Qwen/Qwen3-235B-A22B", "Qwen/Qwen3-VL-235B-A22B-Instruct", "MiniMax/MiniMax-M2.7:MiniMax"] },
    { source: "GPT Image / Gemini", capability: "image", models: ["gpt-image-2", "gpt-image-2-1k", "gpt-image-2-2k", "gpt-image-2-4k", "gpt-image2-2k", "gpt-image2-4k", "nano-banana", "nano-banana-pro", "Nano-Banana-2-2k", "Nano-Banana-2-4k", "Nano-Banana-Pro-2k", "Nano-Banana-Pro-4k", "gemini-3.1-flash-image-preview", "gemini-3-pro-image-preview"] },
    { source: "Grok 图像", capability: "image", models: ["grok-imagine-image", "grok-imagine-image-pro", "grok-imagine-image-edit"] },
    { source: "ModelScope", capability: "image", models: ["Tongyi-MAI/Z-Image-Turbo", "Qwen/Qwen-Image-2512", "Qwen/Qwen-Image-Edit-2511", "black-forest-labs/FLUX.2-klein-9B"] },
    { source: "RunningHub", capability: "image", models: ["gpt-image-2.0/text-to-image-channel-low-price", "gpt-image-2.0/edit-channel-low-price", "gpt-image-2/text-to-image-official-stable", "gpt-image-2/image-to-image-official-stable", "nano-banana/text-to-image-official-stable", "nano-banana/edit-official-stable"] },
    { source: "即梦", capability: "image", models: ["5.0Pro", "5.0", "4.7", "4.6", "4.5", "4.1", "4.0", "3.1", "3.0"] },
    { source: "Agnes AI", capability: "image", models: ["agnes-image-2.1-flash", "agnes-image-2.0-flash"] },
    { source: "Veo", capability: "video", models: ["veo2", "veo2-fast", "veo2-pro", "veo3", "veo3-fast", "veo3-pro", "veo3.1", "veo3.1-fast", "veo3.1-quality", "veo3.1-lite"] },
    { source: "Sora", capability: "video", models: ["sora2", "sora-2", "sora-2-pro"] },
    { source: "Seedance", capability: "video", models: ["seedance-2.0-fast", "seedance2.0_vip", "seedance2.0fast_vip", "seedance2.0", "seedance2.0fast", "seedance2.0mini", "doubao-seedance-2-0-260128", "doubao-seedance-2-0-fast-260128", "doubao-seedance-1-5-pro-251215", "doubao-seedance-1-0-pro-250528", "doubao-seedance-1-0-lite-t2v-250428", "doubao-seedance-1-0-lite-i2v-250428"] },
    { source: "Wan", capability: "video", models: ["wan2.6-t2v", "wan2.6-i2v", "wan2.5-t2v-preview", "wan2.5-i2v-preview", "wan2.2-t2v-plus", "wan2.2-i2v-plus", "wan2.2-i2v-flash"] },
    { source: "其他视频模型", capability: "video", models: ["grok-imagine-video", "grok-imagine-video-1.5", "kling-v3", "pixverse-v6", "agnes-video-v2.0"] },
    { source: "RunningHub", capability: "video", models: ["google/veo3.1-fast/text-to-video-channel-low-price", "sora-2/text-to-video-official-stable", "seedance-2.0-global/text-to-video", "seedance-2.0-global/image-to-video"] },
];

export const referenceModelCatalog: ReferenceCatalogModel[] = groups.flatMap((group) =>
    group.models.map((model) => ({ model, capability: group.capability, source: group.source })),
);
