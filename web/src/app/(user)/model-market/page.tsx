"use client";

import { App, Button, Empty, Input, Tag } from "antd";
import { AudioLines, Flame, ImageIcon, Search, Settings2, Sparkles, UserRound, Video, Wrench } from "lucide-react";
import { useRouter } from "next/navigation";
import { useEffect, useMemo, useState } from "react";

import { fetchModelMarket, type MarketModelCard } from "@/services/api/model-market";
import { formatCNY } from "@/constant/credits";
import { useConfigStore } from "@/stores/use-config-store";
import { useUserStore } from "@/stores/use-user-store";

const categories = [
    { value: "all", label: "全部", icon: <Sparkles className="size-4" /> },
    { value: "hot", label: "热门", icon: <Flame className="size-4" /> },
    { value: "image", label: "图片", icon: <ImageIcon className="size-4" /> },
    { value: "video", label: "视频", icon: <Video className="size-4" /> },
    { value: "person", label: "人物", icon: <UserRound className="size-4" /> },
    { value: "audio", label: "音频", icon: <AudioLines className="size-4" /> },
    { value: "tool", label: "工具", icon: <Wrench className="size-4" /> },
] as const;

export default function ModelMarketPage() {
    const { message } = App.useApp();
    const router = useRouter();
    const [items, setItems] = useState<MarketModelCard[]>([]);
    const [category, setCategory] = useState<(typeof categories)[number]["value"]>("all");
    const [query, setQuery] = useState("");
    const [loading, setLoading] = useState(true);
    const updateConfig = useConfigStore((state) => state.updateConfig);
    const user = useUserStore((state) => state.user);

    useEffect(() => {
        void fetchModelMarket().then(setItems).catch((error) => message.error(error instanceof Error ? error.message : "模型广场加载失败")).finally(() => setLoading(false));
    }, [message]);

    const visible = useMemo(() => {
        const keyword = query.trim().toLowerCase();
        return items.filter((item) => (category === "all" || (category === "hot" ? item.featured : item.category === category)) && (!keyword || `${item.name} ${item.description} ${item.id}`.toLowerCase().includes(keyword)));
    }, [category, items, query]);

    const useModel = (item: MarketModelCard) => {
        if (!item.available) { message.info("模型卡片已经建立，等待管理员配置主线路和备用线路"); return; }
        if (item.category === "image") updateConfig("imageModel", item.id);
        else if (item.category === "audio") updateConfig("audioModel", item.id);
        else updateConfig("videoModel", item.id);
        updateConfig("model", item.id);
        router.push(`/canvas?marketModel=${encodeURIComponent(item.id)}`);
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
                    const price = item.prices[0];
                    return <article key={item.id} className="rounded-2xl border border-stone-200 p-5 dark:border-stone-800">
                        <div className="flex items-start justify-between gap-3"><div><h2 className="font-semibold">{item.name}</h2><p className="mt-1 text-xs text-stone-500">{item.id}</p></div><Tag color={item.status === "normal" ? "green" : item.status === "busy" ? "orange" : "red"}>{item.status === "normal" ? "正常" : item.status === "busy" ? "拥堵" : "维护"}</Tag></div>
                        <p className="mt-4 min-h-10 text-sm text-stone-500">{item.description}</p>
                        <div className="mt-4 flex flex-wrap gap-1.5">{item.resolutions.map((value) => <Tag key={value}>{value}</Tag>)}{item.durations.map((value) => <Tag key={value}>{value} 秒</Tag>)}{item.supportsPerson ? <Tag color="purple">人物参考</Tag> : null}{item.supportsFirstLastFrame ? <Tag>首尾帧</Tag> : null}</div>
                        <div className="mt-5 flex items-end justify-between gap-3"><div><div className="text-xs text-stone-500">售价</div><div className="mt-1 text-lg font-semibold">{price ? `${formatCNY(price.priceCredits)} / ${price.unit}` : "待定价"}</div></div><Button type="primary" disabled={!item.available} onClick={() => useModel(item)}>{item.available ? "立即使用" : "待接入"}</Button></div>
                    </article>;
                })}</div>}
            </div>
        </main>
    );
}
