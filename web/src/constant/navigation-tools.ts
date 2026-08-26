import { FileText, ImagePlus, Images, Library, Maximize2, Store, Video } from "lucide-react";

export const navigationTools = [
    {
        slug: "canvas",
        label: "我的画布",
        icon: Maximize2,
    },
    {
        slug: "image",
        label: "生图工作台",
        icon: ImagePlus,
    },
    {
        slug: "video",
        label: "视频创作台",
        icon: Video,
    },
    {
        slug: "model-market",
        label: "模型广场",
        icon: Store,
    },
    {
        slug: "prompts",
        label: "提示词库",
        icon: FileText,
    },
    {
        slug: "asset-library",
        label: "素材库",
        icon: Library,
    },
    {
        slug: "assets",
        label: "我的素材",
        icon: Images,
    },
] as const;

export type NavigationToolSlug = (typeof navigationTools)[number]["slug"];

