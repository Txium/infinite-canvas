"use client";

import { App, Button, Empty, Input, Select, Tag } from "antd";
import { AudioLines, Box, Flame, ImageIcon, MessageSquare, Music2, Search, Settings2, Sparkles, UserRound, Video, Wrench } from "lucide-react";
import { useRouter } from "next/navigation";
import { useEffect, useMemo, useState } from "react";

import { fetchModelMarket, type MarketModelCard, type MarketModelVariant } from "@/services/api/model-market";
import { formatCNY } from "@/constant/credits";
import { useConfigStore } from "@/stores/use-config-store";
import { useUserStore } from "@/stores/use-user-store";

const categories = [
    { value: "all", label: "全部", icon: <Sparkles className="size-4" /> },
    { value: "hot", label: "热门", icon: <Flame className="size-4" /> },
    { value: "llm", label: "大语言模型", icon: <MessageSquare className="size-4" /> },
    { value: "image", label: "图片", icon: <ImageIcon className="size-4" /> },
    { value: "video", label: "视频", icon: <Video className="size-4" /> },
    { value: "person", label: "人物", icon: <UserRound className="size-4" /> },
    { value: "music", label: "音乐", icon: <Music2 className="size-4" /> },
    { value: "voice", label: "语音", icon: <AudioLines className="size-4" /> },
    { value: "3d", label: "3D", icon: <Box className="size-4" /> },
    { value: "tool", label: "工具", icon: <Wrench className="size-4" /> },
] as const;

export default function ModelMarketPage() {
    const { message } = App.useApp();
    const router = useRouter();
    const [items, setItems] = useState<MarketModelCard[]>([]);
    const [category, setCategory] = useState<(typeof categories)[number]["value"]>("all");
    const [query, setQuery] = useState("");
    const [loading, setLoading] = useState(true);
    const [selectedVariants, setSelectedVariants] = useState<Record<string, string>>({});
    const updateConfig = useConfigStore((state) => state.updateConfig);
    const user = useUserStore((state) => state.user);

    useEffect(() => {
        void fetchModelMarket().then(setItems).catch((error) => message.error(error instanceof Error ? error.message : "模型广场加载失败")).finally(() => setLoading(false));
    }, [message]);

    const visible = useMemo(() => {
        const keyword = query.trim().toLowerCase();
        return items.filter((item) => (category === "all" || (category === "hot" ? item.featured : item.category === category)) && (!keyword || `${item.name} ${item.description} ${item.id}`.toLowerCase().includes(keyword)));
    }, [category, items, query]);

    const selectedVariant = (item: MarketModelCard) => item.variants.find((variant) => variant.id === selectedVariants[item.id]) || item.variants[0];
    const useModel = (item: MarketModelCard, variant: MarketModelVariant | undefined) => {
        if (!variant || !item.availableVariantIds.includes(variant.id)) { message.info("这个档位等待管理员配置并启用 API 线路"); return; }
        if (item.category === "image") updateConfig("imageModel", variant.id);
        else if (item.category === "music" || item.category === "voice") updateConfig("audioModel", variant.id);
        else if (item.category === "llm") updateConfig("textModel", variant.id);
        else updateConfig("videoModel", variant.id);
        updateConfig("model", variant.id);
        router.push(`/canvas?marketModel=${encodeURIComponent(item.id)}&marketVariant=${encodeURIComponent(variant.id)}`);
    };

    return (
        <main className="h-full min-h-0 overflow-y-auto px-4 py-6 md:px-8">
            <div className="mx-auto max-w-[1500px]">
                <section className="mb-6 flex flex-col gap-4 rounded-2xl border border-stone-200 p-6 dark:border-stone-800 md:flex-row md:items-end md:justify-between">
                    <div><h1 className="text-3xl font-semibold">模型广场</h1><p className="mt-2 text-sm text-stone-500">统一模型名称，后台自动选择主线路与备用线路；用户不会看到供应商。</p></div>
                    {user?.role === "admin" ? <Button icon={<Settings2 className="size-4" />} onClick={() => router.push("/admin/model-routing")}>管理模型线路</Button> : null}
                </section>
                <div className="mb-5 flex flex-col gap-3 rounded-2xl border border-stone-200 p-4 dark:border-stone-800 md:flex-row md:items-center md:justify-between">
                    <div className="flex flex-wrap gap-2">{categories.map((item) => <Button key={item.value} type={category === item.value ? "primary" : "default"} icon={item.icon} onClick={() => setCategory(item.value)}>{item.label}</Button>)}</div>
                    <Input allowClear value={query} onChange={(event) => setQuery(event.target.value)} prefix={<Search className="size-4" />} placeholder="搜索模型" className="md:max-w-xs" />
                </div>
                {!loading && visible.length === 0 ? <Empty description="暂无模型" /> : <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-4">{visible.map((item) => {
                    const variant = selectedVariant(item);
                    const variantAvailable = Boolean(variant && item.availableVariantIds.includes(variant.id));
                    return <article key={item.id} className="rounded-2xl border border-stone-200 p-5 dark:border-stone-800">
                        <div className="flex items-start justify-between gap-3"><h2 className="font-semibold">{item.name}</h2><Tag color={item.status === "normal" ? "green" : item.status === "busy" ? "orange" : "red"}>{item.status === "normal" ? "正常" : item.status === "busy" ? "拥堵" : "维护"}</Tag></div>
                        <p className="mt-4 min-h-10 text-sm text-stone-500">{item.description}</p>
                        <div className="mt-4 flex flex-wrap gap-1.5">{item.resolutions.map((value) => <Tag key={value}>{value}</Tag>)}{item.durations.map((value) => <Tag key={value}>{value} 秒</Tag>)}{item.supportsPerson ? <Tag color="purple">人物参考</Tag> : null}{item.supportsFirstLastFrame ? <Tag>首尾帧</Tag> : null}</div>
                        <Select className="mt-4 w-full" value={variant?.id} onChange={(value) => setSelectedVariants((current) => ({...current,[item.id]:value}))} options={item.variants.map((option) => ({value:option.id,label:option.name,disabled:!option.enabled}))} placeholder="选择档位" />
                        <div className="mt-5 flex items-end justify-between gap-3"><div><div className="text-xs text-stone-500">售价</div><div className="mt-1 text-sm font-semibold">{variant ? variant.pricingMode === "fixed" && typeof variant.priceCents === "number" ? `${formatCNY(variant.priceCents)} ${variant.billingUnit}` : variant.pricingMode === "dynamic" ? variant.priceFormula : "暂不上架" : "待定价"}</div></div><Button type="primary" disabled={!variantAvailable} onClick={() => useModel(item,variant)}>{variantAvailable ? "立即使用" : "待接入"}</Button></div>
                    </article>;
                })}</div>}
            </div>
        </main>
    );
}
